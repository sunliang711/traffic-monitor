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
			config.ProvideSSHConfig,
			config.ProvideSecurityConfig,
			config.ProvideBootstrapConfig,
			logger.NewLogger,
			bootstrap.NewDB,
			bootstrap.NewGinEngine,
			bootstrap.NewHTTPServer,
			bootstrap.NewSessionStore,
			bootstrap.NewDataProtector,
			bootstrap.NewSSHRunner,
			repo.NewAdminRepo,
			repo.NewHealthRepo,
			repo.NewMachineRepo,
			repo.NewSSHKeyRepo,
			repo.NewThresholdRuleRepo,
			repo.NewTrafficSampleRepo,
			service.NewAuthService,
			service.NewHealthService,
			service.NewMachineService,
			service.NewSSHKeyService,
			service.NewThresholdService,
			service.NewTrafficCollectionService,
			middleware.NewAuthMiddleware,
			handler.NewAuthHandler,
			handler.NewHealthHandler,
			handler.NewMachineHandler,
			handler.NewSSHKeyHandler,
			handler.NewThresholdHandler,
			handler.NewTrafficSampleHandler,
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
