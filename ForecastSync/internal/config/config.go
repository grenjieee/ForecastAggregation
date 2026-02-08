package config

import (
	"fmt"
	"time"

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
	BaseURL    string  `mapstructure:"base_url"`    // API基础地址
	Protocol   string  `mapstructure:"protocol"`    // 协议类型：rest/ws
	Timeout    int     `mapstructure:"timeout"`     // 请求超时（秒）
	RetryCount int     `mapstructure:"retry_count"` // 重试次数
	SportPath  string  `mapstructure:"sport_path"`  // 体育事件接口路径
	AuthToken  string  `mapstructure:"auth_token"`  // 通用认证Token
	AuthKey    string  `mapstructure:"auth_key"`    // Kalshi专属API Key
	AuthSecret string  `mapstructure:"auth_secret"` // Kalshi专属API Secret
	Proxy      string  `mapstructure:"proxy"`       // 代理地址
	MinBet     float64 `mapstructure:"min_bet"`     // 最小下注金额
	MaxBet     float64 `mapstructure:"max_bet"`     // 最大下注金额
}

// LoadConfig 加载配置文件（默认路径：config/config.yaml）
func LoadConfig() (*Config, error) {
	// 设置配置文件规则
	viper.SetConfigName("config")   // 文件名：config.yaml
	viper.SetConfigType("yaml")     // 格式：yaml
	viper.AddConfigPath("./config") // 路径：项目根目录/config/

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析到结构体（注意：Duration类型需要viper自动解析）
	viper.SetTypeByDefaultValue(true) // 启用默认类型解析（如Duration）
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	return &cfg, nil
}

// GetGORMConfig GetMySQLConfig 获取MySQL配置（适配GORM）
func (m *MySQLConfig) GetGORMConfig() gorm.Config {
	return gorm.Config{} // 可扩展：添加日志、命名策略等
}
