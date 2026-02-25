package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v4/stdlib"

	"ForecastSync/internal/adapter/kalshi"
	"ForecastSync/internal/adapter/polymarket"
	"ForecastSync/internal/api"
	"ForecastSync/internal/config"
	"ForecastSync/internal/interfaces"
	"ForecastSync/internal/listener"
	"ForecastSync/internal/model"
	"ForecastSync/internal/repository"
	"ForecastSync/internal/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// initLogger 根据配置初始化 logrus：可选文件输出、按大小切割、按天数归档。
// 默认 10MB 切割、保留 2 天；file_path 为空则仅 stdout。
func initLogger(cfg *config.Config) *logrus.Logger {
	l := logrus.New()
	l.SetLevel(logrus.InfoLevel)

	lc := &cfg.Log
	if lc.FilePath == "" {
		l.SetOutput(os.Stdout)
		return l
	}

	dir := filepath.Dir(lc.FilePath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			l.SetOutput(os.Stdout)
			l.Warnf("创建日志目录失败 %s，回退到 stdout: %v", dir, err)
			return l
		}
	}

	rotator := &lumberjack.Logger{
		Filename:   lc.FilePath,
		MaxSize:    lc.MaxSizeMB,
		MaxAge:     lc.MaxAgeDays,
		MaxBackups: 0,
		Compress:   false,
	}
	if lc.AlsoStdout {
		l.SetOutput(io.MultiWriter(rotator, os.Stdout))
	} else {
		l.SetOutput(rotator)
	}
	return l
}

// ensureDatabaseExists 当目标库不存在时，连接到 postgres 默认库并创建目标库（幂等）。
// dsn 须为 URL 形式，如 postgres://user:pass@host:port/dbname?options
func ensureDatabaseExists(dsn string) error {
	u, err := url.Parse(dsn)
	if err != nil {
		return err
	}
	dbname := strings.TrimPrefix(u.Path, "/")
	if idx := strings.Index(dbname, "?"); idx >= 0 {
		dbname = dbname[:idx]
	}
	dbname = strings.TrimSpace(dbname)
	if dbname == "" || dbname == "postgres" {
		return nil
	}
	u.Path = "/postgres"
	adminDSN := u.String()
	db, err := sql.Open("pgx", adminDSN)
	if err != nil {
		return err
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {

		}
	}(db)
	err = db.QueryRow("SELECT 1 FROM pg_database WHERE datname = $1", dbname).Scan(new(int))
	if errors.Is(err, sql.ErrNoRows) {
		_, err = db.Exec("CREATE DATABASE " + `"` + strings.ReplaceAll(dbname, `"`, `""`) + `"`)
		return err
	}
	return err
}

func main() {
	// 1. 加载配置文件
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}

	// 2. 初始化日志（路径、轮转、归档均从 config 读取，默认 10MB 切割、保留 2 天）
	logrusLogger := initLogger(cfg)
	logrusLogger.Info("配置文件加载成功")

	// 3. 初始化GORM日志器（修正：正确创建GORM默认日志器）
	// 核心修正：logger.Default() 是方法，不是变量！
	gormLogger := logger.Default.LogMode(logger.Info) // 显示SQL日志（Info级别）

	// 4. 初始化 PostgreSQL 连接（库不存在则先创建再连）
	db, err := gorm.Open(postgres.Open(cfg.MySQL.DSN), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "3D000") {
			logrusLogger.Info("目标数据库不存在，尝试自动创建…")
			if e := ensureDatabaseExists(cfg.MySQL.DSN); e != nil {
				logrusLogger.Fatalf("创建数据库失败: %v", e)
			}
			db, err = gorm.Open(postgres.Open(cfg.MySQL.DSN), &gorm.Config{Logger: gormLogger})
		}
		if err != nil {
			logrusLogger.Fatalf("连接PostgreSQL失败: %v", err)
		}
	}
	logrusLogger.Info("PostgreSQL连接成功")

	// 5. 配置PostgreSQL连接池（复用原有逻辑，参数通用）
	sqlDB, err := db.DB()
	if err != nil {
		logrusLogger.Fatalf("获取SQL DB失败: %v", err)
	}
	// 从配置读取连接池参数（如果cfg中没有PG专属参数，可复用MySQL的，或新增）
	sqlDB.SetMaxOpenConns(cfg.MySQL.MaxOpenConns)       // 也可新增 cfg.PostgreSQL.MaxOpenConns
	sqlDB.SetMaxIdleConns(cfg.MySQL.MaxIdleConns)       // 同上
	sqlDB.SetConnMaxLifetime(cfg.MySQL.ConnMaxLifetime) // 同上

	// 6. 库表不存在则自动创建（按依赖顺序迁移）
	if err := db.AutoMigrate(
		&model.User{},
		&model.Platform{},
		&model.Event{},
		&model.EventOdds{},
		&model.Order{},
		&model.ContractEvent{},
		&model.SettlementRecord{},
		&model.CanonicalEvent{},
		&model.EventPlatformLink{},
	); err != nil {
		logrusLogger.Fatalf("数据库表结构迁移失败: %v", err)
	}
	logrusLogger.Info("数据库表结构检查完成（不存在则已创建）")

	// 7. 配置Gin运行模式（从配置读取：debug/release）
	gin.SetMode(cfg.Server.Mode)
	r := gin.Default()

	// CORS：允许前端跨域请求（开发默认 localhost:3000）
	origins := cfg.Server.CORSAllowOrigins
	if len(origins) == 0 {
		origins = []string{"http://localhost:3000", "http://127.0.0.1:3000"}
	}
	r.Use(cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	// 注册ppof 方便调试和监测性能问题
	pprof.Register(r)
	logrusLogger.Infof("Gin运行模式: %s", cfg.Server.Mode)

	// 8. 注册API路由（传入全局配置）
	syncHandler := api.NewSyncHandler(db, logrusLogger, cfg)
	r.POST("/sync/platform/:platform", syncHandler.SyncPlatformHandler)

	// 市场查询接口（给前端页面用）
	marketHandler := api.NewMarketHandler(db, logrusLogger)
	r.GET("/api/markets", marketHandler.ListMarkets)
	r.GET("/api/markets/:event_uuid", marketHandler.GetMarketDetail)

	// 订单查询与下单接口（注入 Kalshi/Polymarket 测试环境适配器）
	tradingAdapters := map[uint64]interfaces.TradingAdapter{
		1: polymarket.NewTradingAdapter(cfg),
		2: kalshi.NewTradingAdapter(cfg),
	}
	orderHandler := api.NewOrderHandler(db, logrusLogger, tradingAdapters, cfg)
	r.GET("/api/orders", orderHandler.ListOrders)
	r.POST("/api/orders/prepare", orderHandler.PrepareOrder)
	r.POST("/api/orders/place", orderHandler.PlaceOrder)
	r.GET("/api/orders/:order_uuid", orderHandler.GetOrderDetail)
	r.GET("/api/orders/:order_uuid/withdraw-info", orderHandler.GetWithdrawInfo)
	r.POST("/api/orders/:order_uuid/withdraw", orderHandler.RequestWithdraw)
	r.POST("/api/orders/unfreeze", orderHandler.RequestUnfreeze)
	r.GET("/api/orders/contract-order-status", orderHandler.GetContractOrderStatus)

	// 9. 链上事件监听（Escrow FundsLocked → DepositSuccess；Settlement Settled → OnSettlementCompleted）
	orderSvcForListener := service.NewOrderService(db, logrusLogger, tradingAdapters)
	contractListener := listener.NewContractListener(orderSvcForListener, cfg, logrusLogger)
	go func() {
		if err := contractListener.Start(context.Background()); err != nil {
			logrusLogger.WithError(err).Warn("ContractListener exited")
		}
	}()

	// 10. 定时赔率同步
	if cfg.Sync.OddsSyncEnabled && cfg.Sync.OddsSyncIntervalSec > 0 {
		interval := time.Duration(cfg.Sync.OddsSyncIntervalSec) * time.Second
		eventRepo := repository.NewEventRepositoryInstance(db)
		marketRepo := repository.NewMarketRepository(db)
		liveOddsFetchers := make(map[uint64]interfaces.LiveOddsFetcher)
		if p, ok := cfg.Platforms["polymarket"]; ok {
			if lf, ok := polymarket.NewPolymarketAdapter(&p, logrusLogger).(interfaces.LiveOddsFetcher); ok {
				liveOddsFetchers[1] = lf
			}
		}
		if k, ok := cfg.Platforms["kalshi"]; ok {
			if lf, ok := kalshi.NewKalshiAdapter(&k, logrusLogger).(interfaces.LiveOddsFetcher); ok {
				liveOddsFetchers[2] = lf
			}
		}
		oddsSync := service.NewOddsSyncService(marketRepo, eventRepo, liveOddsFetchers, logrusLogger)
		go func() {
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for range ticker.C {
				if err := oddsSync.Run(context.Background(), 500); err != nil {
					logrusLogger.WithError(err).Warn("OddsSync Run failed")
				}
			}
		}()
		logrusLogger.Infof("OddsSync 已启动，间隔 %v", interval)
	}

	// 11. 启动服务
	port := cfg.Server.Port
	logrusLogger.Infof("服务启动成功，端口：%d", port)
	if err := r.Run(fmt.Sprintf(":%d", port)); err != nil {
		logrusLogger.Fatalf("启动服务失败: %v", err)
	}
}
