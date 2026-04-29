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
			result, err := authService.EnsureBootstrapAdmin(startContext)
			if err != nil {
				return fmt.Errorf("ensure bootstrap admin: %w", err)
			}

			if result.Created {
				if result.GeneratedPassword {
					log.Warn().
						Str("username", result.Username).
						Str("password", result.Password).
						Msg("bootstrap admin created with generated password; change it immediately after first login")
					return nil
				}

				log.Info().Str("username", result.Username).Msg("bootstrap admin created")
				return nil
			}

			log.Info().Msg("bootstrap admin initialization skipped")
			return nil
		},
	})
}
