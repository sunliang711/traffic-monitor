package config

import (
	"fmt"
	"time"
)

type Config struct {
	App       AppConfig       `mapstructure:"app"`
	HTTP      HTTPConfig      `mapstructure:"http"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Log       LogConfig       `mapstructure:"log"`
	Session   SessionConfig   `mapstructure:"session"`
	SSH       SSHConfig       `mapstructure:"ssh"`
	Security  SecurityConfig  `mapstructure:"security"`
	Bootstrap BootstrapConfig `mapstructure:"bootstrap"`
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

type SessionConfig struct {
	Secret     string        `mapstructure:"secret"`
	CookieName string        `mapstructure:"cookie_name"`
	MaxAge     time.Duration `mapstructure:"max_age"`
	Secure     bool          `mapstructure:"secure"`
}

type SecurityConfig struct {
	AppMasterKey string `mapstructure:"app_master_key"`
}

type SSHConfig struct {
	DialTimeout    time.Duration `mapstructure:"dial_timeout"`
	CommandTimeout time.Duration `mapstructure:"command_timeout"`
}

type BootstrapConfig struct {
	InitAdminUsername string `mapstructure:"init_admin_username"`
	InitAdminPassword string `mapstructure:"init_admin_password"`
}

func NewConfig() (Config, error) {
	loader := NewLoader(defaultSources())
	return loader.Load()
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

	if cfg.Session.Secret == "" {
		return fmt.Errorf("session.secret is required")
	}

	if cfg.Session.CookieName == "" {
		return fmt.Errorf("session.cookie_name is required")
	}

	if cfg.Session.MaxAge <= 0 {
		return fmt.Errorf("session.max_age must be greater than zero")
	}

	if cfg.SSH.DialTimeout <= 0 {
		return fmt.Errorf("ssh.dial_timeout must be greater than zero")
	}

	if cfg.SSH.CommandTimeout <= 0 {
		return fmt.Errorf("ssh.command_timeout must be greater than zero")
	}

	if cfg.Security.AppMasterKey == "" {
		return fmt.Errorf("security.app_master_key is required")
	}

	if (cfg.Bootstrap.InitAdminUsername == "") != (cfg.Bootstrap.InitAdminPassword == "") {
		return fmt.Errorf("bootstrap init admin username and password must be configured together")
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

func ProvideSessionConfig(cfg Config) SessionConfig {
	return cfg.Session
}

func ProvideSSHConfig(cfg Config) SSHConfig {
	return cfg.SSH
}

func ProvideSecurityConfig(cfg Config) SecurityConfig {
	return cfg.Security
}

func ProvideBootstrapConfig(cfg Config) BootstrapConfig {
	return cfg.Bootstrap
}
