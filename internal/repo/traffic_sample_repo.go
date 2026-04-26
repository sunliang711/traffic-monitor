package repo

import (
	"context"
	"fmt"
	"time"

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

type TrafficSampleDeleteFilter struct {
	MachineID        *uint
	PeriodType       string
	BeforeBucketTime *time.Time
	Limit            int
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
		Order("period_type desc").
		Order("machine_id asc").
		Order("id asc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&samples).Error; err != nil {
		return nil, 0, fmt.Errorf("list traffic samples: %w", err)
	}

	return samples, total, nil
}

func (repo *TrafficSampleRepo) DeleteBefore(ctx context.Context, filter TrafficSampleDeleteFilter) (int64, error) {
	query := repo.db.WithContext(ctx).Model(&model.TrafficSample{})

	if filter.MachineID != nil {
		query = query.Where("machine_id = ?", *filter.MachineID)
	}

	if filter.PeriodType != "" {
		query = query.Where("period_type = ?", filter.PeriodType)
	}

	if filter.BeforeBucketTime != nil {
		query = query.Where("bucket_time < ?", *filter.BeforeBucketTime)
	}

	if filter.Limit > 0 {
		var ids []uint
		if err := query.Order("bucket_time asc").Limit(filter.Limit).Pluck("id", &ids).Error; err != nil {
			return 0, fmt.Errorf("list traffic sample ids for delete: %w", err)
		}

		if len(ids) == 0 {
			return 0, nil
		}

		result := repo.db.WithContext(ctx).Where("id IN ?", ids).Delete(&model.TrafficSample{})
		if result.Error != nil {
			return 0, fmt.Errorf("delete traffic samples by ids: %w", result.Error)
		}

		return result.RowsAffected, nil
	}

	result := query.Delete(&model.TrafficSample{})
	if result.Error != nil {
		return 0, fmt.Errorf("delete traffic samples: %w", result.Error)
	}

	return result.RowsAffected, nil
}
