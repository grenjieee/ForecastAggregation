package main

import (
	"fmt"
	"log"

	"ForecastSync/internal/api"
	"ForecastSync/internal/config"
	_ "ForecastSync/internal/importer" //引入这个保证各个平台方的init能被触发

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger" // 保留logger包，但修正使用方式
)

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

	// 4. 初始化MySQL连接（使用正确的GORM日志配置）
	// 4. 初始化PostgreSQL连接（替换MySQL驱动）
	// 注意：PostgreSQL的DSN格式与MySQL不同，需确保cfg中配置的是PG的DSN
	db, err := gorm.Open(postgres.Open(cfg.MySQL.DSN), &gorm.Config{
		Logger: gormLogger, // 使用正确的GORM日志器
	})
	if err != nil {
		logrusLogger.Fatalf("连接PostgreSQL失败: %v", err)
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

	// 6. 配置Gin运行模式（从配置读取：debug/release）
	gin.SetMode(cfg.Server.Mode)
	r := gin.Default()
	logrusLogger.Infof("Gin运行模式: %s", cfg.Server.Mode)

	// 7. 注册API路由（传入全局配置）
	syncHandler := api.NewSyncHandler(db, logrusLogger, cfg)
	r.POST("/sync/platform/:platform", syncHandler.SyncPlatformHandler)

	// 8. 启动服务（从配置读取端口）
	port := cfg.Server.Port
	logrusLogger.Infof("服务启动成功，端口：%d", port)
	if err := r.Run(fmt.Sprintf(":%d", port)); err != nil {
		logrusLogger.Fatalf("启动服务失败: %v", err)
	}
}
