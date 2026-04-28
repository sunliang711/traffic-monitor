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

type ThresholdRuleDimension struct {
	PeriodType string
	MetricType string
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

func (repo *ThresholdRuleRepo) ReplaceMachineRules(ctx context.Context, machineID uint, rules []model.MachineThresholdRule, inheritedDimensions []ThresholdRuleDimension) error {
	if len(rules) == 0 && len(inheritedDimensions) == 0 {
		return nil
	}

	if err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := &ThresholdRuleRepo{db: tx}
		if err := txRepo.DeleteMachineRules(ctx, machineID, inheritedDimensions); err != nil {
			return err
		}

		if err := txRepo.UpsertMachineRules(ctx, rules); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return fmt.Errorf("replace machine threshold rules: %w", err)
	}

	return nil
}

func (repo *ThresholdRuleRepo) DeleteMachineRules(ctx context.Context, machineID uint, dimensions []ThresholdRuleDimension) error {
	if len(dimensions) == 0 {
		return nil
	}

	dimensionQuery := repo.db.Where("period_type = ? AND metric_type = ?", dimensions[0].PeriodType, dimensions[0].MetricType)
	for _, dimension := range dimensions[1:] {
		dimensionQuery = dimensionQuery.Or("period_type = ? AND metric_type = ?", dimension.PeriodType, dimension.MetricType)
	}

	query := repo.db.WithContext(ctx).
		Where("machine_id = ?", machineID).
		Where(dimensionQuery)

	if err := query.Delete(&model.MachineThresholdRule{}).Error; err != nil {
		return fmt.Errorf("delete machine threshold rules: %w", err)
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
