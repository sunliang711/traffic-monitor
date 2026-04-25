package service

import (
	"context"
	"testing"

	"traffic-monitor/internal/dto"
	"traffic-monitor/internal/model"

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
