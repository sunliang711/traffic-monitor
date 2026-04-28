package config

import (
	"context"
	"strings"

	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

func LogSanitizedConfig(log zerolog.Logger, cfg Config) {
	logger := log.With().Str("component", "config").Logger()

	logger.Info().
		Str("app_name", cfg.App.Name).
		Str("app_env", cfg.App.Env).
		Bool("collector_enabled", cfg.Collector.Enabled).
		Dur("collector_interval", cfg.Collector.Interval).
		Int("collector_max_workers", cfg.Collector.MaxWorkers).
		Int("collector_retry_times", cfg.Collector.RetryTimes).
		Bool("history_cleanup_enabled", cfg.HistoryCleanup.Enabled).
		Dur("history_cleanup_interval", cfg.HistoryCleanup.Interval).
		Int("history_cleanup_samples_days", cfg.HistoryCleanup.SamplesDays).
		Int("history_cleanup_alerts_days", cfg.HistoryCleanup.AlertsDays).
		Int("history_cleanup_batch_size", cfg.HistoryCleanup.BatchSize).
		Dur("history_cleanup_timeout", cfg.HistoryCleanup.Timeout).
		Str("http_addr", cfg.HTTP.Addr).
		Dur("http_read_timeout", cfg.HTTP.ReadTimeout).
		Dur("http_write_timeout", cfg.HTTP.WriteTimeout).
		Dur("http_stop_timeout", cfg.HTTP.StopTimeout).
		Int("database_max_idle_conns", cfg.Database.MaxIdleConns).
		Int("database_max_open_conns", cfg.Database.MaxOpenConns).
		Dur("database_conn_max_lifetime", cfg.Database.ConnMaxLifetime).
		Dur("database_ping_timeout", cfg.Database.PingTimeout).
		Str("log_level", cfg.Log.Level).
		Str("log_format", cfg.Log.Format).
		Str("session_cookie_name", cfg.Session.CookieName).
		Dur("session_max_age", cfg.Session.MaxAge).
		Bool("session_secure", cfg.Session.Secure).
		Bool("session_secret_configured", strings.TrimSpace(cfg.Session.Secret) != "").
		Dur("ssh_dial_timeout", cfg.SSH.DialTimeout).
		Dur("ssh_command_timeout", cfg.SSH.CommandTimeout).
		Bool("security_app_master_key_configured", strings.TrimSpace(cfg.Security.AppMasterKey) != "").
		Bool("bootstrap_admin_enabled", strings.TrimSpace(cfg.Bootstrap.InitAdminUsername) != "" && strings.TrimSpace(cfg.Bootstrap.InitAdminPassword) != "").
		Str("bootstrap_admin_username", maskBootstrapUsername(cfg.Bootstrap.InitAdminUsername)).
		Bool("restore_enabled", cfg.Restore.Enabled()).
		Str("restore_mode", cfg.Restore.Mode).
		Bool("restore_token_configured", strings.TrimSpace(cfg.Restore.Token) != "").
		Msg("sanitized config loaded")
}

func RegisterConfigLogging(lifecycle fx.Lifecycle, log zerolog.Logger, cfg Config) {
	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			LogSanitizedConfig(log, cfg)
			return nil
		},
	})
}

func maskBootstrapUsername(username string) string {
	username = strings.TrimSpace(username)
	if username == "" {
		return ""
	}

	if len(username) <= 2 {
		return username[:1] + "*"
	}

	return username[:1] + strings.Repeat("*", len(username)-2) + username[len(username)-1:]
}
