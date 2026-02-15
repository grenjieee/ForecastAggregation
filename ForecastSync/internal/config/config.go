package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

// Config 全局配置结构体（完全匹配config.yaml）
type Config struct {
	Server    ServerConfig              `mapstructure:"server"`    // 服务器配置
	MySQL     MySQLConfig               `mapstructure:"mysql"`     // MySQL配置
	Sync      SyncConfig                `mapstructure:"sync"`      // 同步调度配置
	Platforms map[string]PlatformConfig `mapstructure:"platforms"` // 多平台独立配置
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port int    `mapstructure:"port"` // 服务端口
	Mode string `mapstructure:"mode"` // Gin运行模式：debug/release/test
}

// MySQLConfig MySQL数据库配置
type MySQLConfig struct {
	DSN             string        `mapstructure:"dsn"`               // 连接DSN
	MaxOpenConns    int           `mapstructure:"max_open_conns"`    // 最大打开连接数
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`    // 最大空闲连接数
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"` // 连接最大存活时间
}

// SyncConfig 同步调度配置
type SyncConfig struct {
	Cron             string   `mapstructure:"cron"`              // 全局同步Cron表达式
	EnabledPlatforms []string `mapstructure:"enabled_platforms"` // 启用的平台列表
}

// PlatformConfig 单个平台的独立配置
type PlatformConfig struct {
	BaseURL      string  `mapstructure:"base_url"`      // API基础地址
	Protocol     string  `mapstructure:"protocol"`      // 协议类型：rest/ws
	Timeout      int     `mapstructure:"timeout"`       // 请求超时（秒）
	RetryCount   int     `mapstructure:"retry_count"`   // 重试次数
	SportPath    string  `mapstructure:"sport_path"`    // 体育事件接口路径（Polymarket 等用）
	SeriesTicker string  `mapstructure:"series_ticker"` // Kalshi 体育系列 ticker，仅拉取体育时填（如 NFL、NBA 等）
	AuthToken    string  `mapstructure:"auth_token"`    // 通用认证Token
	AuthKey      string  `mapstructure:"auth_key"`      // Kalshi专属API Key
	AuthSecret   string  `mapstructure:"auth_secret"`   // Kalshi专属API Secret
	Proxy        string  `mapstructure:"proxy"`         // 代理地址
	MinBet       float64 `mapstructure:"min_bet"`       // 最小下注金额
	MaxBet       float64 `mapstructure:"max_bet"`       // 最大下注金额
}

// LoadConfig 加载配置文件（config/config.yaml），敏感项从 .env 覆盖（不提交 git）
func LoadConfig() (*Config, error) {
	// 1. 加载 .env（若存在），env 中的值会覆盖 config.yaml 中同名字段
	_ = godotenv.Load() // 忽略错误（.env 可不存在）

	// 2. 读取 config.yaml
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	viper.SetTypeByDefaultValue(true)
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 3. 敏感字段：用 env 覆盖（优先级 env > yaml）
	overrideFromEnv(&cfg)
	return &cfg, nil
}

// overrideFromEnv 用环境变量覆盖敏感配置
func overrideFromEnv(cfg *Config) {
	if k, ok := cfg.Platforms["kalshi"]; ok {
		if v := os.Getenv("KALSHI_AUTH_KEY"); v != "" {
			k.AuthKey = v
		}
		if v := os.Getenv("KALSHI_AUTH_SECRET"); v != "" {
			k.AuthSecret = v
		}
		if v := os.Getenv("KALSHI_PROXY"); v != "" {
			k.Proxy = v
		}
		cfg.Platforms["kalshi"] = k
	}
	if p, ok := cfg.Platforms["polymarket"]; ok {
		if v := os.Getenv("POLYMARKET_AUTH_TOKEN"); v != "" {
			p.AuthToken = v
		}
		if v := os.Getenv("POLYMARKET_PROXY"); v != "" {
			p.Proxy = v
		}
		cfg.Platforms["polymarket"] = p
	}
	if v := os.Getenv("MYSQL_DSN"); v != "" {
		cfg.MySQL.DSN = v
	}
}

// GetGORMConfig GetMySQLConfig 获取MySQL配置（适配GORM）
func (m *MySQLConfig) GetGORMConfig() gorm.Config {
	return gorm.Config{} // 可扩展：添加日志、命名策略等
}
