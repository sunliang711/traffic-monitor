package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoaderLoad_MergesSourcesInOrder(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	require.NoError(t, os.MkdirAll(configDir, 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(`
[http]
addr = ":9090"

[database]
dsn = "postgres://from-config"
max_open_conns = 30
`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(configDir, "private.toml"), []byte(`
[database]
dsn = "postgres://from-private"

[log]
level = "debug"
`), 0o644))

	require.NoError(t, os.Setenv("HTTP_ADDR", ":10080"))
	require.NoError(t, os.Setenv("POSTGRES_DSN", "postgres://from-env"))
	require.NoError(t, os.Setenv("SESSION_SECRET", "test-session-secret"))
	require.NoError(t, os.Setenv("APP_MASTER_KEY", "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="))
	require.NoError(t, os.Setenv("SSH_DIAL_TIMEOUT", "7s"))
	require.NoError(t, os.Setenv("COLLECTOR_INTERVAL", "90s"))
	require.NoError(t, os.Setenv("RESTORE_MODE", "admin-password"))
	require.NoError(t, os.Setenv("RESTORE_TOKEN", "0123456789abcdef0123456789abcdef"))
	t.Cleanup(func() {
		_ = os.Unsetenv("HTTP_ADDR")
		_ = os.Unsetenv("POSTGRES_DSN")
		_ = os.Unsetenv("SESSION_SECRET")
		_ = os.Unsetenv("APP_MASTER_KEY")
		_ = os.Unsetenv("SSH_DIAL_TIMEOUT")
		_ = os.Unsetenv("COLLECTOR_INTERVAL")
		_ = os.Unsetenv("RESTORE_MODE")
		_ = os.Unsetenv("RESTORE_TOKEN")
	})

	loader := NewLoader([]Source{
		NewEmbeddedFileSource("embedded-default", defaultConfigFS, "default.toml"),
		NewFileSource("config-file", filepath.Join(configDir, "config.toml"), true),
		NewFileSource("private-file", filepath.Join(configDir, "private.toml"), true),
		NewEnvSource("environment", []EnvBinding{
			{Key: "http.addr", EnvName: "HTTP_ADDR"},
			{Key: "database.dsn", EnvName: "POSTGRES_DSN"},
			{Key: "session.secret", EnvName: "SESSION_SECRET"},
			{Key: "ssh.dial_timeout", EnvName: "SSH_DIAL_TIMEOUT"},
			{Key: "security.app_master_key", EnvName: "APP_MASTER_KEY"},
			{Key: "restore.mode", EnvName: "RESTORE_MODE"},
			{Key: "restore.token", EnvName: "RESTORE_TOKEN"},
		}),
	})

	cfg, err := loader.Load()
	require.NoError(t, err)
	require.Equal(t, "traffic-monitor", cfg.App.Name)
	require.Equal(t, ":10080", cfg.HTTP.Addr)
	require.Equal(t, "postgres://from-env", cfg.Database.DSN)
	require.Equal(t, 90*time.Second, cfg.Collector.Interval)
	require.Equal(t, 30, cfg.Database.MaxOpenConns)
	require.Equal(t, "debug", cfg.Log.Level)
	require.Equal(t, 10*time.Second, cfg.HTTP.ReadTimeout)
	require.Equal(t, "test-session-secret", cfg.Session.Secret)
	require.Equal(t, 7*time.Second, cfg.SSH.DialTimeout)
	require.Equal(t, "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=", cfg.Security.AppMasterKey)
	require.Equal(t, "admin-password", cfg.Restore.Mode)
	require.True(t, cfg.Restore.Enabled())
}

func TestLoaderLoad_ReturnsValidationErrorWhenRequiredValueMissing(t *testing.T) {
	loader := NewLoader([]Source{
		NewEmbeddedFileSource("embedded-default", defaultConfigFS, "default.toml"),
	})

	_, err := loader.Load()
	require.Error(t, err)
	require.ErrorContains(t, err, "database.dsn is required")
}

func TestConfigValidate_ReturnsErrorWhenRestoreModeTokenMissing(t *testing.T) {
	cfg := validTestConfig()
	cfg.Restore.Mode = RestoreModeAdminPassword
	cfg.Restore.Token = ""

	err := cfg.Validate()
	require.Error(t, err)
	require.ErrorContains(t, err, "restore.token")
}

func TestConfigValidate_ReturnsErrorWhenRestoreModeInvalid(t *testing.T) {
	cfg := validTestConfig()
	cfg.Restore.Mode = "enabled"
	cfg.Restore.Token = "0123456789abcdef0123456789abcdef"

	err := cfg.Validate()
	require.Error(t, err)
	require.ErrorContains(t, err, "restore.mode")
}

func validTestConfig() Config {
	return Config{
		Collector: CollectorConfig{
			Enabled:    true,
			Interval:   time.Minute,
			MaxWorkers: 1,
		},
		HistoryCleanup: HistoryCleanupConfig{
			Enabled:     true,
			Interval:    time.Hour,
			SamplesDays: 1,
			AlertsDays:  1,
			BatchSize:   1,
			Timeout:     time.Second,
		},
		HTTP: HTTPConfig{
			Addr:         ":8086",
			ReadTimeout:  time.Second,
			WriteTimeout: time.Second,
			StopTimeout:  time.Second,
		},
		Database: DatabaseConfig{
			DSN:             "postgres://test",
			MaxOpenConns:    1,
			ConnMaxLifetime: time.Minute,
			PingTimeout:     time.Second,
		},
		Session: SessionConfig{
			Secret:     "test-session-secret",
			CookieName: "traffic_monitor_session",
			MaxAge:     time.Hour,
		},
		SSH: SSHConfig{
			DialTimeout:    time.Second,
			CommandTimeout: time.Second,
		},
		Security: SecurityConfig{
			AppMasterKey: "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=",
		},
	}
}
