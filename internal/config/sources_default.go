package config

import "embed"

//go:embed default.toml
var defaultConfigFS embed.FS

func defaultEnvBindings() []EnvBinding {
	return []EnvBinding{
		{Key: "database.dsn", EnvName: "POSTGRES_DSN"},
		{Key: "security.app_master_key", EnvName: "APP_MASTER_KEY"},
		{Key: "bootstrap.init_admin_username", EnvName: "INIT_ADMIN_USERNAME"},
		{Key: "bootstrap.init_admin_password", EnvName: "INIT_ADMIN_PASSWORD"},
	}
}
