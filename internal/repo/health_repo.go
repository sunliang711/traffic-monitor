package repo

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

type HealthRepo struct {
	db *gorm.DB
}

func NewHealthRepo(db *gorm.DB) *HealthRepo {
	return &HealthRepo{db: db}
}

func (repo *HealthRepo) Ping(ctx context.Context) error {
	database, err := repo.db.DB()
	if err != nil {
		return fmt.Errorf("get sql db: %w", err)
	}

	if err := database.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	return nil
}
