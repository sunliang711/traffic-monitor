package bootstrap

import (
	"context"
	"fmt"

	"traffic-monitor/internal/service"

	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

func RegisterAdminBootstrap(lifecycle fx.Lifecycle, authService *service.AuthService, log zerolog.Logger) {
	lifecycle.Append(fx.Hook{
		OnStart: func(startContext context.Context) error {
			created, err := authService.EnsureBootstrapAdmin(startContext)
			if err != nil {
				return fmt.Errorf("ensure bootstrap admin: %w", err)
			}

			if created {
				log.Info().Msg("bootstrap admin created")
				return nil
			}

			log.Info().Msg("bootstrap admin initialization skipped")
			return nil
		},
	})
}
