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
		}),
	}
}
