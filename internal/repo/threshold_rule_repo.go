package repo

import (
	"context"
	"fmt"

	"traffic-monitor/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ThresholdRuleRepo struct {
	db *gorm.DB
}

func NewThresholdRuleRepo(db *gorm.DB) *ThresholdRuleRepo {
	return &ThresholdRuleRepo{db: db}
}

func (repo *ThresholdRuleRepo) UpsertGlobalRules(ctx context.Context, rules []model.GlobalThresholdRule) error {
	if len(rules) == 0 {
		return nil
	}

	if err := repo.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "period_type"},
			{Name: "metric_type"},
		},
		DoUpdates: clause.AssignmentColumns([]string{"threshold_mb", "enabled", "updated_at"}),
	}).Create(&rules).Error; err != nil {
		return fmt.Errorf("upsert global threshold rules: %w", err)
	}

	return nil
}

func (repo *ThresholdRuleRepo) ListGlobalRules(ctx context.Context) ([]model.GlobalThresholdRule, error) {
	var rules []model.GlobalThresholdRule
	if err := repo.db.WithContext(ctx).Order("period_type asc, metric_type asc").Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("list global threshold rules: %w", err)
	}

	return rules, nil
}

func (repo *ThresholdRuleRepo) UpsertMachineRules(ctx context.Context, rules []model.MachineThresholdRule) error {
	if len(rules) == 0 {
		return nil
	}

	if err := repo.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "machine_id"},
			{Name: "period_type"},
			{Name: "metric_type"},
		},
		DoUpdates: clause.AssignmentColumns([]string{"threshold_mb", "enabled", "updated_at"}),
	}).Create(&rules).Error; err != nil {
		return fmt.Errorf("upsert machine threshold rules: %w", err)
	}

	return nil
}

func (repo *ThresholdRuleRepo) ListMachineRules(ctx context.Context, machineID uint) ([]model.MachineThresholdRule, error) {
	var rules []model.MachineThresholdRule
	if err := repo.db.WithContext(ctx).
		Where("machine_id = ?", machineID).
		Order("period_type asc, metric_type asc").
		Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("list machine threshold rules: %w", err)
	}

	return rules, nil
}
