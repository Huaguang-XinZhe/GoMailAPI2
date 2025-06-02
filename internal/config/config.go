package config

import (
	"log"

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

// LoadConfig 加载配置
func LoadConfig() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./configs")

	// 启用环境变量支持
	viper.AutomaticEnv()
	viper.SetEnvPrefix("GOMAILAPI")

	// 设置默认值
	viper.SetDefault("server.host", "localhost")
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.grpc_port", "50051")

	// 缓存默认值
	viper.SetDefault("cache.type", "local")
	viper.SetDefault("cache.local.size", 1000)
	// viper.SetDefault("cache.local.l1_expiration", "50m")
	viper.SetDefault("cache.redis.enabled", false)
	viper.SetDefault("cache.redis.host", "localhost")
	viper.SetDefault("cache.redis.port", "6379")
	viper.SetDefault("cache.redis.db", 0)

	viper.SetDefault("log.level", "info")

	// webhook 默认值（开发环境使用 ngrok）
	viper.SetDefault("webhook.base_url", "https://8e77-2408-8948-2011-5678-a96a-ba3e-7315-342.ngrok-free.app")

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Config file not found, using defaults: %v", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		log.Fatalf("Unable to decode config: %v", err)
	}

	return &config
}
