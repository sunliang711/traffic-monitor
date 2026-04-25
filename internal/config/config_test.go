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
	t.Cleanup(func() {
		_ = os.Unsetenv("HTTP_ADDR")
		_ = os.Unsetenv("POSTGRES_DSN")
	})

	loader := NewLoader([]Source{
		NewEmbeddedFileSource("embedded-default", defaultConfigFS, "default.toml"),
		NewFileSource("config-file", filepath.Join(configDir, "config.toml"), true),
		NewFileSource("private-file", filepath.Join(configDir, "private.toml"), true),
		NewEnvSource("environment", []EnvBinding{
			{Key: "http.addr", EnvName: "HTTP_ADDR"},
			{Key: "database.dsn", EnvName: "POSTGRES_DSN"},
		}),
	})

	cfg, err := loader.Load()
	require.NoError(t, err)
	require.Equal(t, "traffic-monitor", cfg.App.Name)
	require.Equal(t, ":10080", cfg.HTTP.Addr)
	require.Equal(t, "postgres://from-env", cfg.Database.DSN)
	require.Equal(t, 30, cfg.Database.MaxOpenConns)
	require.Equal(t, "debug", cfg.Log.Level)
	require.Equal(t, 10*time.Second, cfg.HTTP.ReadTimeout)
}

func TestLoaderLoad_ReturnsValidationErrorWhenRequiredValueMissing(t *testing.T) {
	loader := NewLoader([]Source{
		NewEmbeddedFileSource("embedded-default", defaultConfigFS, "default.toml"),
	})

	_, err := loader.Load()
	require.Error(t, err)
	require.ErrorContains(t, err, "database.dsn is required")
}
