package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig      `mapstructure:"app"`
	HTTP     HTTPConfig     `mapstructure:"http"`
	Database DatabaseConfig `mapstructure:"database"`
	Log      LogConfig      `mapstructure:"log"`
}

type AppConfig struct {
	Name string `mapstructure:"name"`
	Env  string `mapstructure:"env"`
}

type HTTPConfig struct {
	Addr         string        `mapstructure:"addr"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	StopTimeout  time.Duration `mapstructure:"stop_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

type DatabaseConfig struct {
	DSN             string        `mapstructure:"dsn"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	PingTimeout     time.Duration `mapstructure:"ping_timeout"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
}

func NewConfig() (Config, error) {
	viperInstance := viper.New()
	viperInstance.SetConfigType("toml")
	viperInstance.AddConfigPath("config")
	viperInstance.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viperInstance.AutomaticEnv()

	setDefaults(viperInstance)
	bindEnv(viperInstance)

	if err := readPrimaryConfig(viperInstance, "config"); err != nil {
		return Config{}, err
	}

	if err := mergeOptionalConfig(viperInstance, "private"); err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := viperInstance.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (cfg Config) Validate() error {
	if cfg.HTTP.Addr == "" {
		return fmt.Errorf("http.addr is required")
	}

	if cfg.Database.DSN == "" {
		return fmt.Errorf("database.dsn is required")
	}

	if cfg.Database.MaxIdleConns < 0 {
		return fmt.Errorf("database.max_idle_conns must be greater than or equal to zero")
	}

	if cfg.Database.MaxOpenConns <= 0 {
		return fmt.Errorf("database.max_open_conns must be greater than zero")
	}

	if cfg.Database.ConnMaxLifetime <= 0 {
		return fmt.Errorf("database.conn_max_lifetime must be greater than zero")
	}

	if cfg.Database.PingTimeout <= 0 {
		return fmt.Errorf("database.ping_timeout must be greater than zero")
	}

	if cfg.HTTP.ReadTimeout <= 0 {
		return fmt.Errorf("http.read_timeout must be greater than zero")
	}

	if cfg.HTTP.WriteTimeout <= 0 {
		return fmt.Errorf("http.write_timeout must be greater than zero")
	}

	if cfg.HTTP.StopTimeout <= 0 {
		return fmt.Errorf("http.stop_timeout must be greater than zero")
	}

	return nil
}

func ProvideAppConfig(cfg Config) AppConfig {
	return cfg.App
}

func ProvideHTTPConfig(cfg Config) HTTPConfig {
	return cfg.HTTP
}

func ProvideDatabaseConfig(cfg Config) DatabaseConfig {
	return cfg.Database
}

func ProvideLogConfig(cfg Config) LogConfig {
	return cfg.Log
}

func setDefaults(viperInstance *viper.Viper) {
	viperInstance.SetDefault("app.name", "traffic-monitor")
	viperInstance.SetDefault("app.env", "development")
	viperInstance.SetDefault("http.addr", ":8080")
	viperInstance.SetDefault("http.read_timeout", "10s")
	viperInstance.SetDefault("http.write_timeout", "10s")
	viperInstance.SetDefault("http.stop_timeout", "15s")
	viperInstance.SetDefault("database.max_idle_conns", 5)
	viperInstance.SetDefault("database.max_open_conns", 20)
	viperInstance.SetDefault("database.conn_max_lifetime", "30m")
	viperInstance.SetDefault("database.ping_timeout", "5s")
	viperInstance.SetDefault("log.level", "info")
}

func bindEnv(viperInstance *viper.Viper) {
	_ = viperInstance.BindEnv("app.env", "APP_ENV")
	_ = viperInstance.BindEnv("http.addr", "HTTP_ADDR")
	_ = viperInstance.BindEnv("database.dsn", "POSTGRES_DSN")
	_ = viperInstance.BindEnv("log.level", "LOG_LEVEL")
}

func readPrimaryConfig(viperInstance *viper.Viper, name string) error {
	viperInstance.SetConfigName(name)
	if err := viperInstance.ReadInConfig(); err != nil {
		_, isConfigNotFound := err.(viper.ConfigFileNotFoundError)
		if isConfigNotFound {
			return nil
		}

		return fmt.Errorf("read %s config: %w", name, err)
	}

	return nil
}

func mergeOptionalConfig(viperInstance *viper.Viper, name string) error {
	viperInstance.SetConfigName(name)
	if err := viperInstance.MergeInConfig(); err != nil {
		_, isConfigNotFound := err.(viper.ConfigFileNotFoundError)
		if isConfigNotFound {
			return nil
		}

		return fmt.Errorf("read %s config: %w", name, err)
	}

	return nil
}
