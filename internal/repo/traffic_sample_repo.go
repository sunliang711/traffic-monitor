package repo

import (
	"context"
	"fmt"

	"traffic-monitor/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TrafficSampleRepo struct {
	db *gorm.DB
}

type TrafficSampleFilter struct {
	MachineID  *uint
	PeriodType string
	Page       int
	PageSize   int
}

func NewTrafficSampleRepo(db *gorm.DB) *TrafficSampleRepo {
	return &TrafficSampleRepo{db: db}
}

func (repo *TrafficSampleRepo) UpsertSamples(ctx context.Context, samples []model.TrafficSample) error {
	if len(samples) == 0 {
		return nil
	}

	if err := repo.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "machine_id"},
			{Name: "period_type"},
			{Name: "bucket_time"},
		},
		DoUpdates: clause.AssignmentColumns([]string{"upload_mb", "download_mb", "total_mb", "raw_payload", "collected_at", "updated_at"}),
	}).Create(&samples).Error; err != nil {
		return fmt.Errorf("upsert traffic samples: %w", err)
	}

	return nil
}

func (repo *TrafficSampleRepo) List(ctx context.Context, filter TrafficSampleFilter) ([]model.TrafficSample, int64, error) {
	query := repo.db.WithContext(ctx).Model(&model.TrafficSample{})
	if filter.MachineID != nil {
		query = query.Where("machine_id = ?", *filter.MachineID)
	}

	if filter.PeriodType != "" {
		query = query.Where("period_type = ?", filter.PeriodType)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count traffic samples: %w", err)
	}

	page := filter.Page
	if page <= 0 {
		page = 1
	}

	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}

	var samples []model.TrafficSample
	if err := query.Order("bucket_time desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&samples).Error; err != nil {
		return nil, 0, fmt.Errorf("list traffic samples: %w", err)
	}

	return samples, total, nil
}
