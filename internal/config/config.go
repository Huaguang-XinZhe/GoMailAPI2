package config

import (
	"log"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// ServerConfig 服务器配置
type ServerConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	GrpcPort int    `mapstructure:"grpc_port"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// LocalCacheConfig 本地缓存配置
type LocalCacheConfig struct {
	Size int `mapstructure:"size"`
	// L1Expiration string `mapstructure:"l1_expiration"`
}

// CacheConfig 缓存配置
type CacheConfig struct {
	Type  string           `mapstructure:"type"`
	Local LocalCacheConfig `mapstructure:"local"`
	Redis RedisConfig      `mapstructure:"redis"`
}

// // GetL1ExpirationDuration 获取 L1 缓存过期时间
// func (c *CacheConfig) GetL1ExpirationDuration() time.Duration {
// 	duration, err := time.ParseDuration(c.Local.L1Expiration)
// 	if err != nil {
// 		return 50 * time.Minute // 默认 50 分钟
// 	}
// 	return duration
// }

// LogConfig 日志配置
type LogConfig struct {
	Level string `mapstructure:"level"`
}

// WebhookConfig webhook 配置
type WebhookConfig struct {
	BaseURL string `mapstructure:"base_url"`
}

// Config 应用程序完整配置
type Config struct {
	Server  ServerConfig  `mapstructure:"server"`
	Cache   CacheConfig   `mapstructure:"cache"`
	Log     LogConfig     `mapstructure:"log"`
	Webhook WebhookConfig `mapstructure:"webhook"`
}

// IsProduction 检查是否为生产环境
func (c *Config) IsProduction() bool {
	env := strings.ToLower(os.Getenv("GOMAILAPI_ENV"))
	return env == "production" || env == "prod"
}

// LoadConfig 加载配置
func LoadConfig() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./configs")

	// 启用环境变量支持
	viper.AutomaticEnv()
	viper.SetEnvPrefix("GOMAILAPI")

	// 环境变量映射
	viper.BindEnv("server.host", "GOMAILAPI_SERVER_HOST")
	viper.BindEnv("server.port", "GOMAILAPI_SERVER_PORT")
	viper.BindEnv("server.grpc_port", "GOMAILAPI_GRPC_PORT")
	viper.BindEnv("cache.type", "GOMAILAPI_CACHE_TYPE")
	viper.BindEnv("cache.redis.host", "GOMAILAPI_REDIS_HOST")
	viper.BindEnv("cache.redis.port", "GOMAILAPI_REDIS_PORT")
	viper.BindEnv("cache.redis.password", "GOMAILAPI_REDIS_PASSWORD")
	viper.BindEnv("log.level", "GOMAILAPI_LOG_LEVEL")
	viper.BindEnv("webhook.base_url", "GOMAILAPI_WEBHOOK_BASE_URL")

	// 根据环境设置默认值
	isProduction := strings.ToLower(os.Getenv("GOMAILAPI_ENV")) == "production"

	if isProduction {
		// 生产环境默认值
		viper.SetDefault("server.host", "0.0.0.0")
	} else {
		// 开发环境默认值
		viper.SetDefault("server.host", "localhost")
	}

	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.grpc_port", "50051")

	// 缓存默认值
	viper.SetDefault("cache.type", "local")
	viper.SetDefault("cache.local.size", 1000)
	viper.SetDefault("cache.redis.host", "localhost")
	viper.SetDefault("cache.redis.port", "6379")
	viper.SetDefault("cache.redis.db", 0)
	viper.SetDefault("log.level", "info")

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Config file not found, using defaults: %v", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		log.Fatalf("Unable to decode config: %v", err)
	}

	return &config
}
