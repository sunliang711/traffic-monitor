package repo

import (
	"context"
	"fmt"
	"time"

	"traffic-monitor/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AlertRepo struct {
	db *gorm.DB
}

type AlertFilter struct {
	MachineID  *uint
	PeriodType string
	Page       int
	PageSize   int
}

type AlertDeleteFilter struct {
	MachineID        *uint
	BeforeBucketTime time.Time
	Limit            int
}

func NewAlertRepo(db *gorm.DB) *AlertRepo {
	return &AlertRepo{db: db}
}

func (repo *AlertRepo) CreateIfAbsent(ctx context.Context, alert *model.Alert) (bool, error) {
	result := repo.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "alert_key"}},
		DoNothing: true,
	}).Create(alert)
	if result.Error != nil {
		return false, fmt.Errorf("create alert if absent: %w", result.Error)
	}

	return result.RowsAffected > 0, nil
}

func (repo *AlertRepo) UpdateNotifyStatus(ctx context.Context, alertID uint, notifyStatus string) error {
	if err := repo.db.WithContext(ctx).
		Model(&model.Alert{}).
		Where("id = ?", alertID).
		Update("notify_status", notifyStatus).Error; err != nil {
		return fmt.Errorf("update alert notify status: %w", err)
	}

	return nil
}

func (repo *AlertRepo) DeleteExpired(ctx context.Context, filter AlertDeleteFilter) (int64, error) {
	if filter.BeforeBucketTime.IsZero() {
		return 0, fmt.Errorf("before bucket time is required")
	}

	query := repo.db.WithContext(ctx).Model(&model.Alert{}).Where("bucket_time < ?", filter.BeforeBucketTime)
	if filter.MachineID != nil {
		query = query.Where("machine_id = ?", *filter.MachineID)
	}

	if filter.Limit > 0 {
		subQuery := repo.db.WithContext(ctx).
			Model(&model.Alert{}).
			Select("id").
			Where("bucket_time < ?", filter.BeforeBucketTime).
			Order("bucket_time asc").
			Limit(filter.Limit)
		if filter.MachineID != nil {
			subQuery = subQuery.Where("machine_id = ?", *filter.MachineID)
		}

		result := repo.db.WithContext(ctx).Where("id IN (?)", subQuery).Delete(&model.Alert{})
		if result.Error != nil {
			return 0, fmt.Errorf("delete expired alerts: %w", result.Error)
		}

		return result.RowsAffected, nil
	}

	result := query.Delete(&model.Alert{})
	if result.Error != nil {
		return 0, fmt.Errorf("delete expired alerts: %w", result.Error)
	}

	return result.RowsAffected, nil
}

func (repo *AlertRepo) List(ctx context.Context, filter AlertFilter) ([]model.Alert, int64, error) {
	query := repo.db.WithContext(ctx).Model(&model.Alert{})
	if filter.MachineID != nil {
		query = query.Where("machine_id = ?", *filter.MachineID)
	}

	if filter.PeriodType != "" {
		query = query.Where("period_type = ?", filter.PeriodType)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count alerts: %w", err)
	}

	page := filter.Page
	if page <= 0 {
		page = 1
	}

	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}

	var alerts []model.Alert
	if err := query.Order("bucket_time desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&alerts).Error; err != nil {
		return nil, 0, fmt.Errorf("list alerts: %w", err)
	}

	return alerts, total, nil
}
