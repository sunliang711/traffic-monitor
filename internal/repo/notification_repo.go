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

type NotificationProxyRepo struct {
	db *gorm.DB
}

type NotificationDeliveryRepo struct {
	db *gorm.DB
}

func NewNotificationChannelRepo(db *gorm.DB) *NotificationChannelRepo {
	return &NotificationChannelRepo{db: db}
}

func NewNotificationProxyRepo(db *gorm.DB) *NotificationProxyRepo {
	return &NotificationProxyRepo{db: db}
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

func (repo *NotificationProxyRepo) Create(ctx context.Context, notificationProxy *model.NotificationProxy) error {
	if err := repo.db.WithContext(ctx).Create(notificationProxy).Error; err != nil {
		return fmt.Errorf("create notification proxy: %w", err)
	}

	return nil
}

func (repo *NotificationProxyRepo) Update(ctx context.Context, notificationProxy *model.NotificationProxy) error {
	result := repo.db.WithContext(ctx).Model(&model.NotificationProxy{}).
		Where("id = ?", notificationProxy.ID).
		Updates(map[string]interface{}{
			"name":       notificationProxy.Name,
			"proxy_type": notificationProxy.ProxyType,
			"url":        notificationProxy.URL,
		})
	if result.Error != nil {
		return fmt.Errorf("update notification proxy: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (repo *NotificationProxyRepo) Delete(ctx context.Context, proxyID uint) error {
	result := repo.db.WithContext(ctx).Delete(&model.NotificationProxy{}, proxyID)
	if result.Error != nil {
		return fmt.Errorf("delete notification proxy: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (repo *NotificationProxyRepo) GetByID(ctx context.Context, proxyID uint) (*model.NotificationProxy, error) {
	var notificationProxy model.NotificationProxy
	if err := repo.db.WithContext(ctx).Where("id = ?", proxyID).First(&notificationProxy).Error; err != nil {
		return nil, fmt.Errorf("get notification proxy by id: %w", err)
	}

	return &notificationProxy, nil
}

func (repo *NotificationProxyRepo) List(ctx context.Context) ([]model.NotificationProxy, error) {
	var notificationProxies []model.NotificationProxy
	if err := repo.db.WithContext(ctx).Order("id asc").Find(&notificationProxies).Error; err != nil {
		return nil, fmt.Errorf("list notification proxies: %w", err)
	}

	return notificationProxies, nil
}

func (repo *NotificationDeliveryRepo) Create(ctx context.Context, delivery *model.NotificationDelivery) error {
	if err := repo.db.WithContext(ctx).Create(delivery).Error; err != nil {
		return fmt.Errorf("create notification delivery: %w", err)
	}

	return nil
}

func (repo *NotificationDeliveryRepo) GetLatestByAlertIDs(ctx context.Context, alertIDs []uint) (map[uint]model.NotificationDelivery, error) {
	result := make(map[uint]model.NotificationDelivery)
	if len(alertIDs) == 0 {
		return result, nil
	}

	var deliveries []model.NotificationDelivery
	if err := repo.db.WithContext(ctx).
		Where("alert_id IN ?", alertIDs).
		Order("alert_id asc, created_at desc, id desc").
		Find(&deliveries).Error; err != nil {
		return nil, fmt.Errorf("list notification deliveries by alert ids: %w", err)
	}

	for _, delivery := range deliveries {
		if _, exists := result[delivery.AlertID]; exists {
			continue
		}
		result[delivery.AlertID] = delivery
	}

	return result, nil
}
