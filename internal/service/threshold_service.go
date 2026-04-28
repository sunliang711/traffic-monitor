package service

import (
	"context"
	"errors"
	"fmt"
	"math"

	"traffic-monitor/internal/dto"
	"traffic-monitor/internal/model"
	"traffic-monitor/internal/repo"
)

const (
	thresholdPeriodHourly     = "hourly"
	thresholdPeriodDaily      = "daily"
	thresholdMetricUpload     = "upload"
	thresholdMetricDownload   = "download"
	thresholdMetricTotal      = "total"
	thresholdUnitMB           = "MB"
	thresholdUnitGB           = "GB"
	thresholdSourceGlobal     = "global"
	thresholdSourceMachine    = "machine"
	thresholdStrategyInherit  = "inherit"
	thresholdStrategyOverride = "override"
	thresholdStrategyDisabled = "disabled"
)

var (
	ErrInvalidThresholdRule = errors.New("invalid threshold rule")
)

type ThresholdRuleStore interface {
	UpsertGlobalRules(ctx context.Context, rules []model.GlobalThresholdRule) error
	ListGlobalRules(ctx context.Context) ([]model.GlobalThresholdRule, error)
	ReplaceMachineRules(ctx context.Context, machineID uint, rules []model.MachineThresholdRule, inheritedDimensions []repo.ThresholdRuleDimension) error
	ListMachineRules(ctx context.Context, machineID uint) ([]model.MachineThresholdRule, error)
}

type ThresholdMachineStore interface {
	GetByID(ctx context.Context, machineID uint) (*model.Machine, error)
}

type ThresholdService struct {
	thresholdRuleStore ThresholdRuleStore
	machineStore       ThresholdMachineStore
}

func NewThresholdService(thresholdRuleStore *repo.ThresholdRuleRepo, machineStore *repo.MachineRepo) *ThresholdService {
	return &ThresholdService{
		thresholdRuleStore: thresholdRuleStore,
		machineStore:       machineStore,
	}
}

func (service *ThresholdService) UpsertGlobalRules(ctx context.Context, req dto.UpsertThresholdRulesReq) error {
	rules, err := buildGlobalRules(req.Rules)
	if err != nil {
		return err
	}

	if err := service.thresholdRuleStore.UpsertGlobalRules(ctx, rules); err != nil {
		return fmt.Errorf("upsert global threshold rules: %w", err)
	}

	return nil
}

func (service *ThresholdService) ListGlobalRules(ctx context.Context) ([]dto.ThresholdRuleResp, error) {
	rules, err := service.thresholdRuleStore.ListGlobalRules(ctx)
	if err != nil {
		return nil, fmt.Errorf("list global threshold rules: %w", err)
	}

	return toThresholdResponsesFromGlobal(rules), nil
}

func (service *ThresholdService) UpsertMachineRules(ctx context.Context, machineID uint, req dto.UpsertThresholdRulesReq) error {
	if _, err := service.machineStore.GetByID(ctx, machineID); err != nil {
		if repo.IsRecordNotFound(err) {
			return ErrMachineNotFound
		}

		return fmt.Errorf("get machine for threshold upsert: %w", err)
	}

	rules, inheritedDimensions, err := buildMachineRules(machineID, req.Rules)
	if err != nil {
		return err
	}

	if err := service.thresholdRuleStore.ReplaceMachineRules(ctx, machineID, rules, inheritedDimensions); err != nil {
		return fmt.Errorf("replace machine threshold rules: %w", err)
	}

	return nil
}

func (service *ThresholdService) ListEffectiveMachineRules(ctx context.Context, machineID uint) ([]dto.ThresholdRuleResp, error) {
	if _, err := service.machineStore.GetByID(ctx, machineID); err != nil {
		if repo.IsRecordNotFound(err) {
			return nil, ErrMachineNotFound
		}

		return nil, fmt.Errorf("get machine for threshold list: %w", err)
	}

	globalRules, err := service.thresholdRuleStore.ListGlobalRules(ctx)
	if err != nil {
		return nil, fmt.Errorf("list global threshold rules for machine: %w", err)
	}

	machineRules, err := service.thresholdRuleStore.ListMachineRules(ctx, machineID)
	if err != nil {
		return nil, fmt.Errorf("list machine threshold rules: %w", err)
	}

	return mergeEffectiveRules(globalRules, machineRules), nil
}

func buildGlobalRules(payloads []dto.ThresholdRulePayload) ([]model.GlobalThresholdRule, error) {
	rules := make([]model.GlobalThresholdRule, 0, len(payloads))
	for _, payload := range payloads {
		thresholdMB, err := normalizeThresholdMB(payload.ThresholdValue, payload.ThresholdUnit)
		if err != nil {
			return nil, err
		}

		if err := validateThresholdDimension(payload.PeriodType, payload.MetricType); err != nil {
			return nil, err
		}

		rules = append(rules, model.GlobalThresholdRule{
			PeriodType:  payload.PeriodType,
			MetricType:  payload.MetricType,
			ThresholdMB: thresholdMB,
			Enabled:     payload.Enabled,
		})
	}

	return rules, nil
}

func buildMachineRules(machineID uint, payloads []dto.ThresholdRulePayload) ([]model.MachineThresholdRule, []repo.ThresholdRuleDimension, error) {
	rules := make([]model.MachineThresholdRule, 0, len(payloads))
	inheritedDimensions := make([]repo.ThresholdRuleDimension, 0, len(payloads))
	for _, payload := range payloads {
		if err := validateThresholdDimension(payload.PeriodType, payload.MetricType); err != nil {
			return nil, nil, err
		}

		strategy, err := normalizeMachineThresholdStrategy(payload)
		if err != nil {
			return nil, nil, err
		}

		if strategy == thresholdStrategyInherit {
			inheritedDimensions = append(inheritedDimensions, repo.ThresholdRuleDimension{
				PeriodType: payload.PeriodType,
				MetricType: payload.MetricType,
			})
			continue
		}

		thresholdMB, err := normalizeThresholdMB(payload.ThresholdValue, payload.ThresholdUnit)
		if err != nil {
			return nil, nil, err
		}

		rules = append(rules, model.MachineThresholdRule{
			MachineID:   machineID,
			PeriodType:  payload.PeriodType,
			MetricType:  payload.MetricType,
			ThresholdMB: thresholdMB,
			Enabled:     strategy == thresholdStrategyOverride,
		})
	}

	return rules, inheritedDimensions, nil
}

func normalizeMachineThresholdStrategy(payload dto.ThresholdRulePayload) (string, error) {
	switch payload.Strategy {
	case "":
		if payload.Enabled {
			return thresholdStrategyOverride, nil
		}

		return thresholdStrategyDisabled, nil
	case thresholdStrategyInherit, thresholdStrategyOverride, thresholdStrategyDisabled:
		return payload.Strategy, nil
	default:
		return "", ErrInvalidThresholdRule
	}
}

func normalizeThresholdMB(value float64, unit string) (float64, error) {
	if value <= 0 {
		return 0, ErrInvalidThresholdRule
	}

	switch unit {
	case thresholdUnitMB:
		return value, nil
	case thresholdUnitGB:
		return value * 1024, nil
	default:
		return 0, ErrInvalidThresholdRule
	}
}

func validateThresholdDimension(periodType string, metricType string) error {
	if periodType != thresholdPeriodHourly && periodType != thresholdPeriodDaily {
		return ErrInvalidThresholdRule
	}

	switch metricType {
	case thresholdMetricUpload, thresholdMetricDownload, thresholdMetricTotal:
		return nil
	default:
		return ErrInvalidThresholdRule
	}
}

func toThresholdResponsesFromGlobal(rules []model.GlobalThresholdRule) []dto.ThresholdRuleResp {
	result := make([]dto.ThresholdRuleResp, 0, len(rules))
	for _, rule := range rules {
		result = append(result, thresholdRuleResp(rule.PeriodType, rule.MetricType, rule.ThresholdMB, rule.Enabled, thresholdSourceGlobal, ""))
	}

	return result
}

func mergeEffectiveRules(globalRules []model.GlobalThresholdRule, machineRules []model.MachineThresholdRule) []dto.ThresholdRuleResp {
	result := make([]dto.ThresholdRuleResp, 0, len(globalRules)+len(machineRules))
	machineRuleMap := make(map[string]model.MachineThresholdRule, len(machineRules))
	for _, rule := range machineRules {
		machineRuleMap[thresholdRuleKey(rule.PeriodType, rule.MetricType)] = rule
	}

	mergedRuleMap := make(map[string]struct{}, len(globalRules)+len(machineRules))
	for _, globalRule := range globalRules {
		key := thresholdRuleKey(globalRule.PeriodType, globalRule.MetricType)
		mergedRuleMap[key] = struct{}{}
		if machineRule, ok := machineRuleMap[key]; ok {
			strategy := thresholdStrategyDisabled
			if machineRule.Enabled {
				strategy = thresholdStrategyOverride
			}
			result = append(result, thresholdRuleResp(machineRule.PeriodType, machineRule.MetricType, machineRule.ThresholdMB, machineRule.Enabled, thresholdSourceMachine, strategy))
			continue
		}

		result = append(result, thresholdRuleResp(globalRule.PeriodType, globalRule.MetricType, globalRule.ThresholdMB, globalRule.Enabled, thresholdSourceGlobal, thresholdStrategyInherit))
	}

	for _, machineRule := range machineRules {
		key := thresholdRuleKey(machineRule.PeriodType, machineRule.MetricType)
		if _, ok := mergedRuleMap[key]; ok {
			continue
		}

		strategy := thresholdStrategyDisabled
		if machineRule.Enabled {
			strategy = thresholdStrategyOverride
		}
		result = append(result, thresholdRuleResp(machineRule.PeriodType, machineRule.MetricType, machineRule.ThresholdMB, machineRule.Enabled, thresholdSourceMachine, strategy))
	}

	return result
}

func thresholdRuleResp(periodType string, metricType string, thresholdMB float64, enabled bool, source string, strategy string) dto.ThresholdRuleResp {
	value, unit := displayThresholdValue(thresholdMB)
	return dto.ThresholdRuleResp{
		PeriodType:     periodType,
		MetricType:     metricType,
		ThresholdMB:    thresholdMB,
		ThresholdValue: value,
		ThresholdUnit:  unit,
		Enabled:        enabled,
		Source:         source,
		Strategy:       strategy,
	}
}

func displayThresholdValue(thresholdMB float64) (float64, string) {
	if math.Mod(thresholdMB, 1024) == 0 {
		return thresholdMB / 1024, thresholdUnitGB
	}

	return thresholdMB, thresholdUnitMB
}

func thresholdRuleKey(periodType string, metricType string) string {
	return periodType + ":" + metricType
}
