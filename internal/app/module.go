package app

import (
	"traffic-monitor/internal/bootstrap"
	"traffic-monitor/internal/config"
	"traffic-monitor/internal/handler"
	"traffic-monitor/internal/logger"
	"traffic-monitor/internal/middleware"
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
			config.ProvideSessionConfig,
			config.ProvideBootstrapConfig,
			logger.NewLogger,
			bootstrap.NewDB,
			bootstrap.NewGinEngine,
			bootstrap.NewHTTPServer,
			bootstrap.NewSessionStore,
			repo.NewAdminRepo,
			repo.NewHealthRepo,
			service.NewAuthService,
			service.NewHealthService,
			middleware.NewAuthMiddleware,
			handler.NewAuthHandler,
			handler.NewHealthHandler,
		),
		fx.Invoke(
			bootstrap.RegisterDatabaseLifecycle,
			bootstrap.RegisterAdminBootstrap,
			bootstrap.RegisterHTTPServer,
			handler.RegisterRoutes,
		),
	)
	application.Run()
}
