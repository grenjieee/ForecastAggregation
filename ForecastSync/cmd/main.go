package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"

	_ "github.com/jackc/pgx/v4/stdlib"

	"ForecastSync/internal/adapter/kalshi"
	"ForecastSync/internal/adapter/polymarket"
	"ForecastSync/internal/api"
	"ForecastSync/internal/config"
	"ForecastSync/internal/interfaces"
	"ForecastSync/internal/model"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

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

	// 2. 初始化日志
	logrusLogger := logrus.New()
	logrusLogger.SetLevel(logrus.InfoLevel)
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
	r.POST("/api/orders/place", orderHandler.PlaceOrder)
	r.GET("/api/orders/:order_uuid", orderHandler.GetOrderDetail)
	r.GET("/api/orders/:order_uuid/withdraw-info", orderHandler.GetWithdrawInfo)
	r.POST("/api/orders/:order_uuid/withdraw", orderHandler.RequestWithdraw)

	// 9. 启动服务（从配置读取端口）
	port := cfg.Server.Port
	logrusLogger.Infof("服务启动成功，端口：%d", port)
	if err := r.Run(fmt.Sprintf(":%d", port)); err != nil {
		logrusLogger.Fatalf("启动服务失败: %v", err)
	}
}
