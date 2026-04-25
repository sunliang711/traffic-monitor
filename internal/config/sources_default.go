package config

import "embed"

//go:embed default.toml
var defaultConfigFS embed.FS

func defaultSources() []Source {
	return []Source{
		NewEmbeddedFileSource("embedded-default", defaultConfigFS, "default.toml"),
		NewFileSource("config-file", "config/config.toml", true),
		NewFileSource("private-file", "config/private.toml", true),
		NewEnvSource("environment", []EnvBinding{
			{Key: "app.env", EnvName: "APP_ENV"},
			{Key: "http.addr", EnvName: "HTTP_ADDR"},
			{Key: "database.dsn", EnvName: "POSTGRES_DSN"},
			{Key: "log.level", EnvName: "LOG_LEVEL"},
			{Key: "log.format", EnvName: "LOG_FORMAT"},
			{Key: "session.secret", EnvName: "SESSION_SECRET"},
			{Key: "session.cookie_name", EnvName: "SESSION_COOKIE_NAME"},
			{Key: "session.secure", EnvName: "SESSION_SECURE"},
			{Key: "session.max_age", EnvName: "SESSION_MAX_AGE"},
			{Key: "ssh.dial_timeout", EnvName: "SSH_DIAL_TIMEOUT"},
			{Key: "ssh.command_timeout", EnvName: "SSH_COMMAND_TIMEOUT"},
			{Key: "security.app_master_key", EnvName: "APP_MASTER_KEY"},
			{Key: "bootstrap.init_admin_username", EnvName: "INIT_ADMIN_USERNAME"},
			{Key: "bootstrap.init_admin_password", EnvName: "INIT_ADMIN_PASSWORD"},
			{Key: "history_cleanup.enabled", EnvName: "HISTORY_CLEANUP_ENABLED"},
			{Key: "history_cleanup.interval", EnvName: "HISTORY_CLEANUP_INTERVAL"},
			{Key: "history_cleanup.samples_days", EnvName: "HISTORY_CLEANUP_SAMPLES_DAYS"},
			{Key: "history_cleanup.alerts_days", EnvName: "HISTORY_CLEANUP_ALERTS_DAYS"},
			{Key: "history_cleanup.batch_size", EnvName: "HISTORY_CLEANUP_BATCH_SIZE"},
			{Key: "history_cleanup.timeout", EnvName: "HISTORY_CLEANUP_TIMEOUT"},
		}),
	}
}
