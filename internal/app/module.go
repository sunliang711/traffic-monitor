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
	"go.uber.org/fx/fxevent"
)

func Run() {
	application := fx.New(
		fx.Provide(
			config.NewConfig,
			config.ProvideAppConfig,
			config.ProvideCollectorConfig,
			config.ProvideHistoryCleanupConfig,
			config.ProvideHTTPConfig,
			config.ProvideDatabaseConfig,
			config.ProvideLogConfig,
			config.ProvideSessionConfig,
			config.ProvideSSHConfig,
			config.ProvideSecurityConfig,
			config.ProvideBootstrapConfig,
			config.ProvideRestoreConfig,
			logger.NewLogger,
			logger.NewFxLogger,
			bootstrap.NewDB,
			bootstrap.NewGinEngine,
			bootstrap.NewHTTPServer,
			bootstrap.NewSessionStore,
			fx.Annotate(bootstrap.NewDataProtector, fx.As(new(service.SSHKeyProtector))),
			fx.Annotate(bootstrap.NewSSHRunner, fx.As(new(service.SSHCommandRunner))),
			repo.NewAdminRepo,
			repo.NewHealthRepo,
			repo.NewMachineRepo,
			repo.NewSSHKeyRepo,
			repo.NewThresholdRuleRepo,
			repo.NewTrafficSampleRepo,
			repo.NewAlertRepo,
			repo.NewNotificationChannelRepo,
			repo.NewNotificationProxyRepo,
			repo.NewNotificationDeliveryRepo,
			service.NewAuthService,
			service.NewAlertService,
			service.NewHealthService,
			service.NewMachineService,
			service.NewSSHKeyService,
			service.NewBackupService,
			service.NewThresholdService,
			service.NewTrafficAlertEvaluator,
			service.NewTrafficCollectionService,
			service.NewTrafficCollectRunner,
			service.NewTrafficScheduler,
			service.NewHistoryCleanupService,
			service.NewHistoryCleanupRunner,
			service.NewHistoryCleanupScheduler,
			middleware.NewAuthMiddleware,
			handler.NewAuthHandler,
			handler.NewRestoreHandler,
			handler.NewHealthHandler,
			handler.NewMachineHandler,
			handler.NewSSHKeyHandler,
			handler.NewBackupHandler,
			handler.NewThresholdHandler,
			handler.NewTrafficSampleHandler,
			handler.NewAlertHandler,
		),
		fx.WithLogger(func(log logger.FxLogger) fxevent.Logger {
			return log
		}),
		fx.Invoke(
			bootstrap.RegisterDatabaseLifecycle,
			bootstrap.RegisterAdminBootstrap,
			bootstrap.RegisterHTTPServer,
			config.RegisterConfigLogging,
			service.RegisterTrafficScheduler,
			service.RegisterHistoryCleanupScheduler,
			handler.RegisterRoutes,
		),
	)
	application.Run()
}
