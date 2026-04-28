package service

import (
	"context"
	"testing"

	"traffic-monitor/internal/dto"
	"traffic-monitor/internal/model"
	"traffic-monitor/internal/repo"

	"github.com/stretchr/testify/require"
)

type stubThresholdRuleStore struct {
	globalRules  []model.GlobalThresholdRule
	machineRules map[uint][]model.MachineThresholdRule
}

func (store *stubThresholdRuleStore) UpsertGlobalRules(_ context.Context, rules []model.GlobalThresholdRule) error {
	store.globalRules = rules
	return nil
}

func (store *stubThresholdRuleStore) ListGlobalRules(_ context.Context) ([]model.GlobalThresholdRule, error) {
	return store.globalRules, nil
}

func (store *stubThresholdRuleStore) UpsertMachineRules(_ context.Context, rules []model.MachineThresholdRule) error {
	if store.machineRules == nil {
		store.machineRules = make(map[uint][]model.MachineThresholdRule)
	}

	if len(rules) > 0 {
		store.machineRules[rules[0].MachineID] = rules
	}

	return nil
}

func (store *stubThresholdRuleStore) DeleteMachineRules(_ context.Context, machineID uint, dimensions []repo.ThresholdRuleDimension) error {
	if len(dimensions) == 0 {
		return nil
	}

	if store.machineRules == nil {
		return nil
	}

	currentRules := store.machineRules[machineID]
	nextRules := make([]model.MachineThresholdRule, 0, len(currentRules))
	for _, rule := range currentRules {
		shouldDelete := false
		for _, dimension := range dimensions {
			if rule.PeriodType == dimension.PeriodType && rule.MetricType == dimension.MetricType {
				shouldDelete = true
				break
			}
		}
		if !shouldDelete {
			nextRules = append(nextRules, rule)
		}
	}
	store.machineRules[machineID] = nextRules

	return nil
}

func (store *stubThresholdRuleStore) ReplaceMachineRules(ctx context.Context, machineID uint, rules []model.MachineThresholdRule, inheritedDimensions []repo.ThresholdRuleDimension) error {
	if err := store.DeleteMachineRules(ctx, machineID, inheritedDimensions); err != nil {
		return err
	}

	return store.UpsertMachineRules(ctx, rules)
}

func (store *stubThresholdRuleStore) ListMachineRules(_ context.Context, machineID uint) ([]model.MachineThresholdRule, error) {
	return store.machineRules[machineID], nil
}

type stubThresholdMachineStore struct {
	machines map[uint]*model.Machine
}

func (store *stubThresholdMachineStore) GetByID(_ context.Context, machineID uint) (*model.Machine, error) {
	machine, ok := store.machines[machineID]
	if !ok {
		return nil, ErrMachineNotFound
	}

	return machine, nil
}

func TestNormalizeThresholdMB(t *testing.T) {
	value, err := normalizeThresholdMB(2, thresholdUnitGB)
	require.NoError(t, err)
	require.Equal(t, 2048.0, value)

	value, err = normalizeThresholdMB(512, thresholdUnitMB)
	require.NoError(t, err)
	require.Equal(t, 512.0, value)
}

func TestListEffectiveMachineRules(t *testing.T) {
	service := &ThresholdService{
		thresholdRuleStore: &stubThresholdRuleStore{
			globalRules: []model.GlobalThresholdRule{
				{PeriodType: thresholdPeriodHourly, MetricType: thresholdMetricTotal, ThresholdMB: 1024, Enabled: true},
				{PeriodType: thresholdPeriodDaily, MetricType: thresholdMetricTotal, ThresholdMB: 2048, Enabled: true},
			},
			machineRules: map[uint][]model.MachineThresholdRule{
				1: {
					{MachineID: 1, PeriodType: thresholdPeriodHourly, MetricType: thresholdMetricTotal, ThresholdMB: 512, Enabled: true},
				},
			},
		},
		machineStore: &stubThresholdMachineStore{
			machines: map[uint]*model.Machine{
				1: {Base: model.Base{ID: 1}},
			},
		},
	}

	rules, err := service.ListEffectiveMachineRules(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, rules, 2)
	require.Equal(t, thresholdSourceMachine, rules[0].Source)
	require.Equal(t, 512.0, rules[0].ThresholdMB)
	require.Equal(t, thresholdSourceGlobal, rules[1].Source)
}

func TestListEffectiveMachineRulesKeepsDisabledMachineOverride(t *testing.T) {
	service := &ThresholdService{
		thresholdRuleStore: &stubThresholdRuleStore{
			globalRules: []model.GlobalThresholdRule{
				{PeriodType: thresholdPeriodHourly, MetricType: thresholdMetricTotal, ThresholdMB: 1024, Enabled: true},
			},
			machineRules: map[uint][]model.MachineThresholdRule{
				1: {
					{MachineID: 1, PeriodType: thresholdPeriodHourly, MetricType: thresholdMetricTotal, ThresholdMB: 512, Enabled: false},
				},
			},
		},
		machineStore: &stubThresholdMachineStore{
			machines: map[uint]*model.Machine{
				1: {Base: model.Base{ID: 1}},
			},
		},
	}

	rules, err := service.ListEffectiveMachineRules(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, rules, 1)
	require.Equal(t, thresholdSourceMachine, rules[0].Source)
	require.Equal(t, thresholdStrategyDisabled, rules[0].Strategy)
	require.False(t, rules[0].Enabled)
	require.Equal(t, 512.0, rules[0].ThresholdMB)
}

func TestUpsertMachineRulesInheritDeletesMachineOverride(t *testing.T) {
	store := &stubThresholdRuleStore{
		globalRules: []model.GlobalThresholdRule{
			{PeriodType: thresholdPeriodHourly, MetricType: thresholdMetricTotal, ThresholdMB: 1024, Enabled: true},
		},
		machineRules: map[uint][]model.MachineThresholdRule{
			1: {
				{MachineID: 1, PeriodType: thresholdPeriodHourly, MetricType: thresholdMetricTotal, ThresholdMB: 512, Enabled: false},
			},
		},
	}
	service := &ThresholdService{
		thresholdRuleStore: store,
		machineStore: &stubThresholdMachineStore{
			machines: map[uint]*model.Machine{
				1: {Base: model.Base{ID: 1}},
			},
		},
	}

	err := service.UpsertMachineRules(context.Background(), 1, dto.UpsertThresholdRulesReq{
		Rules: []dto.ThresholdRulePayload{
			{
				PeriodType:     thresholdPeriodHourly,
				MetricType:     thresholdMetricTotal,
				ThresholdValue: 0,
				ThresholdUnit:  thresholdUnitGB,
				Strategy:       thresholdStrategyInherit,
			},
		},
	})
	require.NoError(t, err)

	rules, err := service.ListEffectiveMachineRules(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, rules, 1)
	require.Equal(t, thresholdSourceGlobal, rules[0].Source)
	require.Equal(t, thresholdStrategyInherit, rules[0].Strategy)
	require.True(t, rules[0].Enabled)
	require.Equal(t, 1024.0, rules[0].ThresholdMB)
}

func TestUpsertGlobalRulesRejectsInvalidUnit(t *testing.T) {
	service := &ThresholdService{
		thresholdRuleStore: &stubThresholdRuleStore{},
	}

	err := service.UpsertGlobalRules(context.Background(), dto.UpsertThresholdRulesReq{
		Rules: []dto.ThresholdRulePayload{
			{
				PeriodType:     thresholdPeriodHourly,
				MetricType:     thresholdMetricUpload,
				ThresholdValue: 1,
				ThresholdUnit:  "TB",
				Enabled:        true,
			},
		},
	})
	require.ErrorIs(t, err, ErrInvalidThresholdRule)
}
