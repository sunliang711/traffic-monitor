package bootstrap

import (
	"context"
	"database/sql"
	"fmt"

	"traffic-monitor/internal/config"
	"traffic-monitor/internal/model"

	"github.com/rs/zerolog"
	"go.uber.org/fx"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewDB(cfg config.DatabaseConfig) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db: %w", err)
	}

	configureConnectionPool(sqlDB, cfg)

	return db, nil
}

func RegisterDatabaseLifecycle(lifecycle fx.Lifecycle, db *gorm.DB, cfg config.DatabaseConfig, log zerolog.Logger) {
	lifecycle.Append(fx.Hook{
		OnStart: func(startContext context.Context) error {
			pingContext, cancel := context.WithTimeout(startContext, cfg.PingTimeout)
			defer cancel()

			sqlDB, err := db.DB()
			if err != nil {
				return fmt.Errorf("get sql db: %w", err)
			}

			if err := sqlDB.PingContext(pingContext); err != nil {
				return fmt.Errorf("ping database: %w", err)
			}

			if err := db.WithContext(startContext).AutoMigrate(&model.Admin{}, &model.SSHKey{}); err != nil {
				return fmt.Errorf("auto migrate: %w", err)
			}

			log.Info().Msg("database ready")
			return nil
		},
		OnStop: func(stopContext context.Context) error {
			sqlDB, err := db.DB()
			if err != nil {
				return fmt.Errorf("get sql db: %w", err)
			}

			if err := sqlDB.Close(); err != nil {
				return fmt.Errorf("close database: %w", err)
			}

			log.Info().Msg("database closed")
			return nil
		},
	})
}

func configureConnectionPool(sqlDB *sql.DB, cfg config.DatabaseConfig) {
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
}
