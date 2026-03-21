package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	AppEnv         string               `mapstructure:"app_env"`
	Server         ServerConfig         `mapstructure:"server"`
	Database       DatabaseConfig       `mapstructure:"db"`
	Redis          RedisConfig          `mapstructure:"redis"`
	Memcached      MemcachedConfig      `mapstructure:"memcached"`
	JWT            JWTConfig            `mapstructure:"jwt"`
	Cache          CacheConfig          `mapstructure:"cache"`
	Asynq          AsynqConfig          `mapstructure:"asynq"`
	PaymentGateway PaymentGatewayConfig `mapstructure:"payment_gateway"`
}

type ServerConfig struct {
	Port         string        `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Name            string        `mapstructure:"name"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type MemcachedConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Addr    string `mapstructure:"addr"`
}

type JWTConfig struct {
	AccessSecret    string        `mapstructure:"access_secret"`
	RefreshSecret   string        `mapstructure:"refresh_secret"`
	AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`
	Issuer          string        `mapstructure:"issuer"`
	Audience        string        `mapstructure:"audience"`
}

type CacheConfig struct {
	Driver       string        `mapstructure:"driver"`
	DefaultTTL   time.Duration `mapstructure:"default_ttl"`
	UserMeTTL    time.Duration `mapstructure:"user_me_ttl"`
	QueryCacheOn bool          `mapstructure:"query_cache_on"`
}

type AsynqConfig struct {
	Concurrency int `mapstructure:"concurrency"`
}

type PaymentGatewayConfig struct {
	BaseURL        string        `mapstructure:"base_url"`
	Timeout        time.Duration `mapstructure:"timeout"`
	DefaultClient  string        `mapstructure:"default_client"`
	DefaultKey     string        `mapstructure:"default_key"`
	CallbackSecret string        `mapstructure:"callback_secret"`
}

func Load() (Config, error) {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	v := viper.New()
	setDefaults(v, env)

	v.SetConfigFile(fmt.Sprintf(".env.%s", env))
	v.SetConfigType("env")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		_, isNotFound := err.(viper.ConfigFileNotFoundError)
		if !isNotFound {
			if strings.Contains(err.Error(), "no such file") || strings.Contains(err.Error(), "cannot find") {
				// optional env file
			} else {
				return Config{}, fmt.Errorf("read config file: %w", err)
			}
		}
	}

	cfg := Config{}
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}

	if cfg.JWT.AccessSecret == "" || cfg.JWT.RefreshSecret == "" {
		return Config{}, fmt.Errorf("JWT_ACCESS_SECRET and JWT_REFRESH_SECRET must be set")
	}
	if cfg.Database.Password == "" {
		return Config{}, fmt.Errorf("DB_PASSWORD must be set")
	}

	return cfg, nil
}

func setDefaults(v *viper.Viper, env string) {
	v.SetDefault("app_env", env)

	v.SetDefault("server.port", "8080")
	v.SetDefault("server.read_timeout", "10s")
	v.SetDefault("server.write_timeout", "10s")
	v.SetDefault("server.idle_timeout", "60s")

	v.SetDefault("db.host", "127.0.0.1")
	v.SetDefault("db.port", 3306)
	v.SetDefault("db.user", "root")
	v.SetDefault("db.password", "")
	v.SetDefault("db.name", "gue")
	v.SetDefault("db.max_open_conns", 25)
	v.SetDefault("db.max_idle_conns", 25)
	v.SetDefault("db.conn_max_lifetime", "5m")

	v.SetDefault("redis.addr", "127.0.0.1:6379")
	v.SetDefault("redis.db", 0)

	v.SetDefault("memcached.enabled", false)
	v.SetDefault("memcached.addr", "127.0.0.1:11211")

	v.SetDefault("jwt.access_token_ttl", "15m")
	v.SetDefault("jwt.refresh_token_ttl", "168h")
	v.SetDefault("jwt.access_secret", "")
	v.SetDefault("jwt.refresh_secret", "")
	v.SetDefault("jwt.issuer", "gue-starter")
	v.SetDefault("jwt.audience", "gue-clients")

	v.SetDefault("cache.driver", "redis")
	v.SetDefault("cache.default_ttl", "5m")
	v.SetDefault("cache.user_me_ttl", "2m")
	v.SetDefault("cache.query_cache_on", true)

	v.SetDefault("asynq.concurrency", 10)

	v.SetDefault("payment_gateway.base_url", "https://rest.otomatis.vip")
	v.SetDefault("payment_gateway.timeout", "15s")
	v.SetDefault("payment_gateway.default_client", "")
	v.SetDefault("payment_gateway.default_key", "")
	v.SetDefault("payment_gateway.callback_secret", "")
}
