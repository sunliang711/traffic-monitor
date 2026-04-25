package repo

import (
	"context"
	"fmt"

	"traffic-monitor/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type NotificationChannelRepo struct {
	db *gorm.DB
}

type NotificationDeliveryRepo struct {
	db *gorm.DB
}

func NewNotificationChannelRepo(db *gorm.DB) *NotificationChannelRepo {
	return &NotificationChannelRepo{db: db}
}

func NewNotificationDeliveryRepo(db *gorm.DB) *NotificationDeliveryRepo {
	return &NotificationDeliveryRepo{db: db}
}

func (repo *NotificationChannelRepo) Upsert(ctx context.Context, channel *model.NotificationChannel) error {
	if err := repo.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "channel_type"}},
		DoUpdates: clause.AssignmentColumns([]string{"enabled", "config_json", "updated_at"}),
	}).Create(channel).Error; err != nil {
		return fmt.Errorf("upsert notification channel: %w", err)
	}

	return nil
}

func (repo *NotificationChannelRepo) List(ctx context.Context) ([]model.NotificationChannel, error) {
	var channels []model.NotificationChannel
	if err := repo.db.WithContext(ctx).Order("channel_type asc").Find(&channels).Error; err != nil {
		return nil, fmt.Errorf("list notification channels: %w", err)
	}

	return channels, nil
}

func (repo *NotificationDeliveryRepo) Create(ctx context.Context, delivery *model.NotificationDelivery) error {
	if err := repo.db.WithContext(ctx).Create(delivery).Error; err != nil {
		return fmt.Errorf("create notification delivery: %w", err)
	}

	return nil
}
