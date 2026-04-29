package repo

import (
	"context"
	"fmt"

	"traffic-monitor/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AppSettingRepo struct {
	db *gorm.DB
}

func NewAppSettingRepo(db *gorm.DB) *AppSettingRepo {
	return &AppSettingRepo{db: db}
}

func (repo *AppSettingRepo) GetByKey(ctx context.Context, key string) (*model.AppSetting, error) {
	var setting model.AppSetting
	if err := repo.db.WithContext(ctx).Where("setting_key = ?", key).First(&setting).Error; err != nil {
		return nil, fmt.Errorf("get app setting by key: %w", err)
	}

	return &setting, nil
}

func (repo *AppSettingRepo) Upsert(ctx context.Context, setting *model.AppSetting) error {
	if err := repo.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "setting_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"setting_value", "updated_at"}),
	}).Create(setting).Error; err != nil {
		return fmt.Errorf("upsert app setting: %w", err)
	}

	return nil
}
