package app

import (
	"traffic-monitor/internal/bootstrap"
	"traffic-monitor/internal/config"
	"traffic-monitor/internal/handler"
	"traffic-monitor/internal/logger"
	"traffic-monitor/internal/repo"
	"traffic-monitor/internal/service"

	"go.uber.org/fx"
)

func Run() {
	application := fx.New(
		fx.Provide(
			config.NewConfig,
			config.ProvideAppConfig,
			config.ProvideHTTPConfig,
			config.ProvideDatabaseConfig,
			config.ProvideLogConfig,
			logger.NewLogger,
			bootstrap.NewDB,
			bootstrap.NewGinEngine,
			bootstrap.NewHTTPServer,
			repo.NewHealthRepo,
			service.NewHealthService,
			handler.NewHealthHandler,
		),
		fx.Invoke(
			bootstrap.RegisterDatabaseLifecycle,
			bootstrap.RegisterHTTPServer,
			handler.RegisterRoutes,
		),
	)
	application.Run()
}
