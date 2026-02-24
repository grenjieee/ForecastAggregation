package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

// Config 全局配置结构体（完全匹配config.yaml）
type Config struct {
	Server    ServerConfig              `mapstructure:"server"`    // 服务器配置
	MySQL     MySQLConfig               `mapstructure:"mysql"`     // MySQL配置
	Log       LogConfig                 `mapstructure:"log"`       // 日志配置（路径、轮转、归档）
	Sync      SyncConfig                `mapstructure:"sync"`      // 同步调度配置
	Platforms map[string]PlatformConfig `mapstructure:"platforms"` // 多平台独立配置
	Circle    CircleConfig              `mapstructure:"circle"`    // Circle 兑换（占位，后续对接）
	Chain     ChainConfig               `mapstructure:"chain"`     // 链与合约地址（监听与提现）
}

// LogConfig 日志文件与轮转配置
type LogConfig struct {
	// FilePath 日志文件路径；为空则仅输出到 stdout，不写文件
	FilePath string `mapstructure:"file_path"`
	// MaxSizeMB 单文件达到该大小（MB）时切割，默认 10
	MaxSizeMB int `mapstructure:"max_size_mb"`
	// MaxAgeDays 归档保留天数，超过则删除，默认 2
	MaxAgeDays int `mapstructure:"max_age_days"`
	// AlsoStdout 写文件时是否同时输出到 stdout，默认 true
	AlsoStdout bool `mapstructure:"also_stdout"`
}

// ChainConfig 链 RPC 与合约地址（Polymarket 结算、FeeVault 等）
type ChainConfig struct {
	ChainID           int64  `mapstructure:"chain_id"`           // 链 ID，如 137 (Polygon)
	RPCURL            string `mapstructure:"rpc_url"`            // RPC 地址
	WSURL             string `mapstructure:"ws_url"`             // WebSocket 地址（事件订阅）
	EscrowAddress     string `mapstructure:"escrow_address"`     // EscrowVault 合约地址
	BetRouterAddress  string `mapstructure:"bet_router_address"` // BetRouter 合约地址（读 nonce、提交 intent）
	SettlementAddress string `mapstructure:"settlement_address"` // Settlement 合约地址
	FeeVaultAddress   string `mapstructure:"fee_vault_address"`  // FeeVault 合约地址
	// ExecutorPrivateKey 从环境变量 CHAIN_EXECUTOR_PRIVATE_KEY 读取，不写进配置文件
	ExecutorPrivateKey string
}

// CircleConfig Circle API 配置（可配置测试/生产环境）
type CircleConfig struct {
	BaseURL string `mapstructure:"base_url"` // API 地址，如 https://api-sandbox.circle.com
	APIKey  string `mapstructure:"api_key"`  // API Key
	Timeout int    `mapstructure:"timeout"`  // 请求超时（秒）
	Proxy   string `mapstructure:"proxy"`    // 代理地址
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port             int      `mapstructure:"port"`               // 服务端口
	Mode             string   `mapstructure:"mode"`               // Gin运行模式：debug/release/test
	CORSAllowOrigins []string `mapstructure:"cors_allow_origins"` // CORS 允许的 Origin，为空时默认 localhost:3000
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
	Cron                string   `mapstructure:"cron"`                   // 全局同步Cron表达式
	EnabledPlatforms    []string `mapstructure:"enabled_platforms"`      // 启用的平台列表
	OddsSyncIntervalSec int      `mapstructure:"odds_sync_interval_sec"` // 赔率定时同步间隔（秒），如 60
	OddsSyncEnabled     bool     `mapstructure:"odds_sync_enabled"`      // 是否启用定时赔率同步
}

// PlatformConfig 单个平台的独立配置
type PlatformConfig struct {
	BaseURL        string   `mapstructure:"base_url"`         // API基础地址
	Protocol       string   `mapstructure:"protocol"`         // 协议类型：rest/ws
	Timeout        int      `mapstructure:"timeout"`          // 请求超时（秒）
	RetryCount     int      `mapstructure:"retry_count"`      // 重试次数
	SportPath      string   `mapstructure:"sport_path"`       // 体育事件接口路径（Polymarket 等用）
	SeriesTicker   string   `mapstructure:"series_ticker"`    // Kalshi 体育系列 ticker（单个，与 series_tickers 二选一）
	SeriesTickers  []string `mapstructure:"series_tickers"`   // Kalshi 体育系列 ticker 列表，精准拉取时填（如 ["NFL","NBA"]），避免拉取不稳定的 series
	AuthToken      string   `mapstructure:"auth_token"`       // 通用认证Token
	AuthKey        string   `mapstructure:"auth_key"`         // Kalshi API Key；Polymarket CLOB API Key
	AuthSecret     string   `mapstructure:"auth_secret"`      // Kalshi 私钥；Polymarket CLOB API Secret
	AuthPrivateKey string   `mapstructure:"auth_private_key"` // Polymarket 下单用私钥（EIP-712 签名）
	ClobBaseURL    string   `mapstructure:"clob_base_url"`    // Polymarket CLOB 地址（测试/生产均为 clob.polymarket.com）
	Proxy          string   `mapstructure:"proxy"`            // 代理地址
	MinBet         float64  `mapstructure:"min_bet"`          // 最小下注金额
	MaxBet         float64  `mapstructure:"max_bet"`          // 最大下注金额
}

// LoadConfig 加载配置文件（config/config.yaml），敏感项从 .env.local 覆盖（不提交 git）
func LoadConfig() (*Config, error) {
	wd, err := os.Getwd()
	if err != nil {
		println("获取当前目录失败：", err.Error())
	} else {
		println("当前程序运行目录：", wd)
	}
	envPath := filepath.Join(wd, ".env")
	// 1. 加载 .env（若存在），env 中的值会覆盖 config.yaml 中同名字段
	if err := godotenv.Load(envPath); err != nil {
		println("警告：加载根目录.env失败 →", err.Error())
		// 可选：打印.env文件的绝对路径，确认路径是否正确
		println("尝试加载的.env路径：", envPath)
	} else {
		println("✅ 根目录.env文件加载成功")
	}

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

	// 日志默认值：保留 2 天、10MB 切割
	if cfg.Log.MaxSizeMB <= 0 {
		cfg.Log.MaxSizeMB = 10
	}
	if cfg.Log.MaxAgeDays <= 0 {
		cfg.Log.MaxAgeDays = 2
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
		if v := os.Getenv("POLYMARKET_AUTH_KEY"); v != "" {
			p.AuthKey = v
		}
		if v := os.Getenv("POLYMARKET_AUTH_SECRET"); v != "" {
			p.AuthSecret = v
		}
		if v := os.Getenv("POLYMARKET_AUTH_TOKEN"); v != "" {
			p.AuthToken = v
		}
		if v := os.Getenv("POLYMARKET_AUTH_PRIVATE_KEY"); v != "" {
			p.AuthPrivateKey = v
		}
		if v := os.Getenv("POLYMARKET_PROXY"); v != "" {
			p.Proxy = v
		}
		cfg.Platforms["polymarket"] = p
	}
	if v := os.Getenv("MYSQL_DSN"); v != "" {
		cfg.MySQL.DSN = v
	}
	if v := os.Getenv("CIRCLE_API_KEY"); v != "" {
		cfg.Circle.APIKey = v
	}
	if v := os.Getenv("CIRCLE_BASE_URL"); v != "" {
		cfg.Circle.BaseURL = v
	}
	if v := os.Getenv("CHAIN_EXECUTOR_PRIVATE_KEY"); v != "" {
		cfg.Chain.ExecutorPrivateKey = v
	}
}

// GetGORMConfig GetMySQLConfig 获取MySQL配置（适配GORM）
func (m *MySQLConfig) GetGORMConfig() gorm.Config {
	return gorm.Config{} // 可扩展：添加日志、命名策略等
}
