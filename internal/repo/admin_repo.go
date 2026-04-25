package repo

import (
	"context"
	"errors"
	"fmt"

	"traffic-monitor/internal/model"

	"gorm.io/gorm"
)

type AdminRepo struct {
	db *gorm.DB
}

func NewAdminRepo(db *gorm.DB) *AdminRepo {
	return &AdminRepo{db: db}
}

func (repo *AdminRepo) GetByID(ctx context.Context, adminID uint) (*model.Admin, error) {
	var admin model.Admin
	if err := repo.db.WithContext(ctx).Where("id = ?", adminID).First(&admin).Error; err != nil {
		return nil, fmt.Errorf("get admin by id: %w", err)
	}

	return &admin, nil
}

func (repo *AdminRepo) GetByUsername(ctx context.Context, username string) (*model.Admin, error) {
	var admin model.Admin
	if err := repo.db.WithContext(ctx).Where("username = ?", username).First(&admin).Error; err != nil {
		return nil, fmt.Errorf("get admin by username: %w", err)
	}

	return &admin, nil
}

func (repo *AdminRepo) Create(ctx context.Context, admin *model.Admin) error {
	if err := repo.db.WithContext(ctx).Create(admin).Error; err != nil {
		return fmt.Errorf("create admin: %w", err)
	}

	return nil
}

func (repo *AdminRepo) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var count int64
	if err := repo.db.WithContext(ctx).Model(&model.Admin{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return false, fmt.Errorf("count admin by username: %w", err)
	}

	return count > 0, nil
}

func IsRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
