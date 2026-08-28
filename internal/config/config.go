package config

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"
)

const RestoreModeAdminPassword = "admin-password"

const masterKeyBytes = 32

type Config struct {
	App            AppConfig            `mapstructure:"app"`
	Collector      CollectorConfig      `mapstructure:"collector"`
	HistoryCleanup HistoryCleanupConfig `mapstructure:"history_cleanup"`
	HTTP           HTTPConfig           `mapstructure:"http"`
	Database       DatabaseConfig       `mapstructure:"database"`
	Log            LogConfig            `mapstructure:"log"`
	Session        SessionConfig        `mapstructure:"session"`
	SSH            SSHConfig            `mapstructure:"ssh"`
	Security       SecurityConfig       `mapstructure:"security"`
	Bootstrap      BootstrapConfig      `mapstructure:"bootstrap"`
	Restore        RestoreConfig        `mapstructure:"restore"`
}

type AppConfig struct {
	Name string `mapstructure:"name"`
	Env  string `mapstructure:"env"`
}

type CollectorConfig struct {
	Enabled    bool          `mapstructure:"enabled"`
	Interval   time.Duration `mapstructure:"interval"`
	MaxWorkers int           `mapstructure:"max_workers"`
	RetryTimes int           `mapstructure:"retry_times"`
}

type HistoryCleanupConfig struct {
	Enabled     bool          `mapstructure:"enabled"`
	Interval    time.Duration `mapstructure:"interval"`
	SamplesDays int           `mapstructure:"samples_days"`
	AlertsDays  int           `mapstructure:"alerts_days"`
	BatchSize   int           `mapstructure:"batch_size"`
	Timeout     time.Duration `mapstructure:"timeout"`
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
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
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

// MasterKey 解析主密钥，容忍环境变量注入时常见的空白和引号包裹，并兼容 URL 安全和无填充的 base64。
func (cfg SecurityConfig) MasterKey() ([]byte, error) {
	raw := strings.Trim(strings.TrimSpace(cfg.AppMasterKey), "\"'")
	if raw == "" {
		return nil, fmt.Errorf("security.app_master_key is required")
	}

	var decodeErr error
	for _, encoding := range []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	} {
		key, err := encoding.DecodeString(raw)
		if err != nil {
			if decodeErr == nil {
				decodeErr = err
			}

			continue
		}

		if len(key) != masterKeyBytes {
			return nil, fmt.Errorf("security.app_master_key must decode to %d bytes, got %d; generate one with: openssl rand -base64 %d", masterKeyBytes, len(key), masterKeyBytes)
		}

		return key, nil
	}

	return nil, fmt.Errorf("security.app_master_key is not valid base64: %w (length %d after trimming quotes and whitespace); generate one with: openssl rand -base64 %d", decodeErr, len(raw), masterKeyBytes)
}

type SSHConfig struct {
	DialTimeout    time.Duration `mapstructure:"dial_timeout"`
	CommandTimeout time.Duration `mapstructure:"command_timeout"`
}

type BootstrapConfig struct {
	InitAdminUsername string `mapstructure:"init_admin_username"`
	InitAdminPassword string `mapstructure:"init_admin_password"`
}

type RestoreConfig struct {
	Mode  string `mapstructure:"mode"`
	Token string `mapstructure:"token"`
}

func (cfg RestoreConfig) Enabled() bool {
	return cfg.Mode == RestoreModeAdminPassword
}

func NewConfig() (Config, error) {
	return LoadWithSources(
		NewEmbeddedFileSource("config/default.toml", defaultConfigFS, "default.toml"),
		NewFileSource("config/config.toml", "config/config.toml", true),
		NewFileSource("config/private.toml", "config/private.toml", true),
		NewEnvSource("env", defaultEnvBindings()),
	)
}

func (cfg Config) Validate() error {
	if cfg.HTTP.Addr == "" {
		return fmt.Errorf("http.addr is required")
	}

	if cfg.Collector.Interval <= 0 {
		return fmt.Errorf("collector.interval must be greater than zero")
	}

	if cfg.Collector.MaxWorkers <= 0 {
		return fmt.Errorf("collector.max_workers must be greater than zero")
	}

	if cfg.Collector.RetryTimes < 0 {
		return fmt.Errorf("collector.retry_times must be greater than or equal to zero")
	}

	if cfg.HistoryCleanup.Enabled {
		if cfg.HistoryCleanup.Interval <= 0 {
			return fmt.Errorf("history_cleanup.interval must be greater than zero")
		}

		if cfg.HistoryCleanup.SamplesDays <= 0 {
			return fmt.Errorf("history_cleanup.samples_days must be greater than zero")
		}

		if cfg.HistoryCleanup.AlertsDays <= 0 {
			return fmt.Errorf("history_cleanup.alerts_days must be greater than zero")
		}

		if cfg.HistoryCleanup.BatchSize <= 0 {
			return fmt.Errorf("history_cleanup.batch_size must be greater than zero")
		}

		if cfg.HistoryCleanup.Timeout <= 0 {
			return fmt.Errorf("history_cleanup.timeout must be greater than zero")
		}
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

	if _, err := cfg.Security.MasterKey(); err != nil {
		return err
	}

	if (cfg.Bootstrap.InitAdminUsername == "") != (cfg.Bootstrap.InitAdminPassword == "") {
		return fmt.Errorf("bootstrap init admin username and password must be configured together")
	}

	if cfg.Restore.Mode != "" && cfg.Restore.Mode != RestoreModeAdminPassword {
		return fmt.Errorf("restore.mode must be empty or %q", RestoreModeAdminPassword)
	}

	if cfg.Restore.Enabled() && len(strings.TrimSpace(cfg.Restore.Token)) < 32 {
		return fmt.Errorf("restore.token must be configured with at least 32 characters when restore mode is enabled")
	}

	return nil
}

func ProvideAppConfig(cfg Config) AppConfig {
	return cfg.App
}

func ProvideCollectorConfig(cfg Config) CollectorConfig {
	return cfg.Collector
}

func ProvideHistoryCleanupConfig(cfg Config) HistoryCleanupConfig {
	return cfg.HistoryCleanup
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

func ProvideRestoreConfig(cfg Config) RestoreConfig {
	return cfg.Restore
}
