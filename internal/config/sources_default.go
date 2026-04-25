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
			{Key: "session.secret", EnvName: "SESSION_SECRET"},
			{Key: "session.cookie_name", EnvName: "SESSION_COOKIE_NAME"},
			{Key: "session.secure", EnvName: "SESSION_SECURE"},
			{Key: "session.max_age", EnvName: "SESSION_MAX_AGE"},
			{Key: "security.app_master_key", EnvName: "APP_MASTER_KEY"},
			{Key: "bootstrap.init_admin_username", EnvName: "INIT_ADMIN_USERNAME"},
			{Key: "bootstrap.init_admin_password", EnvName: "INIT_ADMIN_PASSWORD"},
		}),
	}
}
