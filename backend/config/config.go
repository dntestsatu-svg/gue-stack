package config

import (
	"fmt"
	"os"
	"path/filepath"
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
	Security       SecurityConfig       `mapstructure:"security"`
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

type SecurityConfig struct {
	Cookie    CookieConfig    `mapstructure:"cookie"`
	CSRF      CSRFConfig      `mapstructure:"csrf"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
}

type CookieConfig struct {
	AccessTokenName  string `mapstructure:"access_token_name"`
	RefreshTokenName string `mapstructure:"refresh_token_name"`
	CSRFCookieName   string `mapstructure:"csrf_cookie_name"`
	Domain           string `mapstructure:"domain"`
	Path             string `mapstructure:"path"`
	Secure           bool   `mapstructure:"secure"`
	HTTPOnly         bool   `mapstructure:"http_only"`
	SameSite         string `mapstructure:"same_site"`
}

type CSRFConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	HeaderName string `mapstructure:"header_name"`
}

type RateLimitConfig struct {
	Enabled       bool          `mapstructure:"enabled"`
	Window        time.Duration `mapstructure:"window"`
	AuthLogin     int64         `mapstructure:"auth_login"`
	AuthRegister  int64         `mapstructure:"auth_register"`
	PaymentBridge int64         `mapstructure:"payment_bridge"`
}

type PaymentGatewayConfig struct {
	BaseURL            string        `mapstructure:"base_url"`
	Timeout            time.Duration `mapstructure:"timeout"`
	DefaultClient      string        `mapstructure:"default_client"`
	DefaultKey         string        `mapstructure:"default_key"`
	MerchantUUID       string        `mapstructure:"merchant_uuid"`
	CallbackSecret     string        `mapstructure:"callback_secret"`
	WebhookSecret      string        `mapstructure:"webhook_secret"`
	PlatformFeePercent int           `mapstructure:"platform_fee_percent"`
}

func Load() (Config, error) {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	v := viper.New()
	setDefaults(v, env)

	v.SetConfigType("env")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if envFilePath := findRootEnvFile(); envFilePath != "" {
		v.SetConfigFile(envFilePath)
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
	cfg.Cache.Driver = strings.ToLower(strings.TrimSpace(cfg.Cache.Driver))
	switch cfg.Cache.Driver {
	case "memcached", "none":
	default:
		return Config{}, fmt.Errorf("CACHE_DRIVER must be one of: memcached, none")
	}
	if cfg.Cache.Driver == "memcached" && (!cfg.Memcached.Enabled || strings.TrimSpace(cfg.Memcached.Addr) == "") {
		return Config{}, fmt.Errorf("CACHE_DRIVER=memcached requires MEMCACHED_ENABLED=true and MEMCACHED_ADDR set")
	}
	if cfg.Security.Cookie.AccessTokenName == "" || cfg.Security.Cookie.RefreshTokenName == "" || cfg.Security.Cookie.CSRFCookieName == "" {
		return Config{}, fmt.Errorf("security cookie names must be set")
	}
	if strings.TrimSpace(cfg.Security.CSRF.HeaderName) == "" {
		return Config{}, fmt.Errorf("SECURITY_CSRF_HEADER_NAME must be set")
	}
	cfg.Security.Cookie.SameSite = strings.ToLower(strings.TrimSpace(cfg.Security.Cookie.SameSite))
	switch cfg.Security.Cookie.SameSite {
	case "strict", "lax", "none":
	default:
		return Config{}, fmt.Errorf("SECURITY_COOKIE_SAME_SITE must be one of: strict, lax, none")
	}
	if cfg.Security.Cookie.SameSite == "none" && !cfg.Security.Cookie.Secure {
		return Config{}, fmt.Errorf("SECURITY_COOKIE_SECURE must be true when SECURITY_COOKIE_SAME_SITE=none")
	}
	if cfg.Security.RateLimit.Window <= 0 {
		return Config{}, fmt.Errorf("SECURITY_RATE_LIMIT_WINDOW must be greater than 0")
	}
	if strings.TrimSpace(cfg.PaymentGateway.MerchantUUID) == "" {
		legacyMerchantUUID := strings.TrimSpace(cfg.PaymentGateway.CallbackSecret)
		if legacyMerchantUUID == "" {
			return Config{}, fmt.Errorf("PAYMENT_GATEWAY_MERCHANT_UUID must be set (or fallback PAYMENT_GATEWAY_CALLBACK_SECRET)")
		}
		cfg.PaymentGateway.MerchantUUID = legacyMerchantUUID
	}
	if cfg.PaymentGateway.PlatformFeePercent < 0 || cfg.PaymentGateway.PlatformFeePercent > 100 {
		return Config{}, fmt.Errorf("PAYMENT_GATEWAY_PLATFORM_FEE_PERCENT must be between 0 and 100")
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

	v.SetDefault("cache.driver", "memcached")
	v.SetDefault("cache.default_ttl", "5m")
	v.SetDefault("cache.user_me_ttl", "2m")
	v.SetDefault("cache.query_cache_on", true)

	v.SetDefault("asynq.concurrency", 10)

	v.SetDefault("security.cookie.access_token_name", "access_token")
	v.SetDefault("security.cookie.refresh_token_name", "refresh_token")
	v.SetDefault("security.cookie.csrf_cookie_name", "csrf_token")
	v.SetDefault("security.cookie.domain", "")
	v.SetDefault("security.cookie.path", "/")
	v.SetDefault("security.cookie.secure", true)
	v.SetDefault("security.cookie.http_only", true)
	v.SetDefault("security.cookie.same_site", "strict")

	v.SetDefault("security.csrf.enabled", true)
	v.SetDefault("security.csrf.header_name", "X-CSRF-Token")

	v.SetDefault("security.rate_limit.enabled", true)
	v.SetDefault("security.rate_limit.window", "1m")
	v.SetDefault("security.rate_limit.auth_login", 10)
	v.SetDefault("security.rate_limit.auth_register", 5)
	v.SetDefault("security.rate_limit.payment_bridge", 60)

	v.SetDefault("payment_gateway.base_url", "https://rest.otomatis.vip")
	v.SetDefault("payment_gateway.timeout", "15s")
	v.SetDefault("payment_gateway.default_client", "")
	v.SetDefault("payment_gateway.default_key", "")
	v.SetDefault("payment_gateway.merchant_uuid", "")
	v.SetDefault("payment_gateway.callback_secret", "")
	v.SetDefault("payment_gateway.webhook_secret", "")
	v.SetDefault("payment_gateway.platform_fee_percent", 3)
}

func findRootEnvFile() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	searchDir := cwd
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(searchDir, ".env")
		info, statErr := os.Stat(candidate)
		if statErr == nil && !info.IsDir() {
			return candidate
		}

		parent := filepath.Dir(searchDir)
		if parent == searchDir {
			break
		}
		searchDir = parent
	}

	return ""
}
