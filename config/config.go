package config

import (
	"bufio"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Env        string `mapstructure:"env"`
	Server     ServerConfig
	Database   DatabaseConfig
	Redis      RedisConfig
	Storage    StorageConfig
	JWT        JWTConfig
	ACME       ACMEConfig
	Log        LogConfig
	Migrations MigrationsConfig
	CORS       CORSConfig
}

type CORSConfig struct {
	AllowOrigins string `mapstructure:"allow_origins"`
}

type ServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	CDNBaseURL   string        `mapstructure:"cdn_base_url"`
}

type DatabaseConfig struct {
	DSN          string `mapstructure:"dsn"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type StorageConfig struct {
	Endpoint      string `mapstructure:"endpoint"`
	Region        string `mapstructure:"region"`
	Bucket        string `mapstructure:"bucket"`
	AccessKey     string `mapstructure:"access_key"`
	SecretKey     string `mapstructure:"secret_key"`
	UsePathStyle  bool   `mapstructure:"use_path_style"`
	PublicBaseURL string `mapstructure:"public_base_url"`
}

type JWTConfig struct {
	AccessSecret  string        `mapstructure:"access_secret"`
	RefreshSecret string        `mapstructure:"refresh_secret"`
	AccessTTL     time.Duration `mapstructure:"access_ttl"`
	RefreshTTL    time.Duration `mapstructure:"refresh_ttl"`
}

type ACMEConfig struct {
	Email   string `mapstructure:"email"`
	Enabled bool   `mapstructure:"enabled"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

type MigrationsConfig struct {
	Path string `mapstructure:"path"`
}

func Load() (*Config, error) {
	// Load .env file into OS environment first so AutomaticEnv picks it up.
	// Keys in .env (e.g. DAM_DATABASE_DSN) become real env vars before Viper reads them.
	loadDotEnv(".env")

	v := viper.New()

	v.SetDefault("env", "development")
	// Server
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", "30s")
	v.SetDefault("server.write_timeout", "30s")
	v.SetDefault("server.cdn_base_url", "")
	// Database — empty defaults so AutomaticEnv resolves them during Unmarshal
	v.SetDefault("database.dsn", "")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	// Redis
	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)
	// Storage
	v.SetDefault("storage.endpoint", "")
	v.SetDefault("storage.region", "us-east-1")
	v.SetDefault("storage.bucket", "")
	v.SetDefault("storage.access_key", "")
	v.SetDefault("storage.secret_key", "")
	v.SetDefault("storage.use_path_style", false)
	v.SetDefault("storage.public_base_url", "")
	// JWT
	v.SetDefault("jwt.access_secret", "")
	v.SetDefault("jwt.refresh_secret", "")
	v.SetDefault("jwt.access_ttl", "15m")
	v.SetDefault("jwt.refresh_ttl", "168h")
	// ACME
	v.SetDefault("acme.email", "")
	v.SetDefault("acme.enabled", false)
	// Logging
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
	// Migrations
	v.SetDefault("migrations.path", "migrations")
	// CORS
	v.SetDefault("cors.allow_origins", "")

	// Optional YAML config file (takes lower priority than env vars)
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("/etc/dam")
	_ = v.ReadInConfig()

	// DAM_DATABASE_DSN → database.dsn, DAM_SERVER_PORT → server.port, etc.
	v.SetEnvPrefix("DAM")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// loadDotEnv parses a .env file and sets any missing keys into the OS environment.
// Existing OS env vars are never overwritten (OS takes precedence).
// Supports: KEY=value, KEY="value", # comments, blank lines.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return // no .env file is fine
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx < 1 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])

		// Strip inline comments
		if ci := strings.Index(val, " #"); ci >= 0 {
			val = strings.TrimSpace(val[:ci])
		}

		// Strip surrounding quotes
		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') ||
				(val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}

		// Only set if not already present in the environment
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, val)
		}
	}
}
