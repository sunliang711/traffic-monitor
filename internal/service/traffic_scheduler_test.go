package service

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"traffic-monitor/internal/config"
	"traffic-monitor/internal/model"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

type stubTrafficCollectRunner struct {
	machines    []model.Machine
	failures    map[uint]int
	calls       sync.Map
	inFlight    atomic.Int32
	maxInFlight atomic.Int32
}

func (runner *stubTrafficCollectRunner) ListEnabledMachines(_ context.Context) ([]model.Machine, error) {
	return runner.machines, nil
}

func (runner *stubTrafficCollectRunner) CollectMachine(_ context.Context, machine *model.Machine) ([]model.TrafficSample, error) {
	current := runner.inFlight.Add(1)
	for {
		maxCurrent := runner.maxInFlight.Load()
		if current <= maxCurrent || runner.maxInFlight.CompareAndSwap(maxCurrent, current) {
			break
		}
	}

	defer runner.inFlight.Add(-1)

	time.Sleep(20 * time.Millisecond)
	callCount := 0
	if value, ok := runner.calls.Load(machine.ID); ok {
		callCount = value.(int)
	}
	callCount++
	runner.calls.Store(machine.ID, callCount)

	if runner.failures[machine.ID] > 0 {
		runner.failures[machine.ID]--
		return nil, errors.New("collect failed")
	}

	return []model.TrafficSample{{MachineID: machine.ID}}, nil
}

func TestTrafficSchedulerRespectsMaxWorkers(t *testing.T) {
	runner := &stubTrafficCollectRunner{
		machines: []model.Machine{
			{Base: model.Base{ID: 1}},
			{Base: model.Base{ID: 2}},
			{Base: model.Base{ID: 3}},
			{Base: model.Base{ID: 4}},
		},
		failures: map[uint]int{},
	}

	scheduler := NewTrafficScheduler(config.CollectorConfig{
		Enabled:    true,
		Interval:   time.Hour,
		MaxWorkers: 2,
		RetryTimes: 0,
	}, runner, zerolog.Nop())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	scheduler.Start(ctx)
	time.Sleep(150 * time.Millisecond)
	scheduler.Stop()
	scheduler.Wait()

	require.LessOrEqual(t, runner.maxInFlight.Load(), int32(2))
}

func TestTrafficSchedulerRetriesFailedJobs(t *testing.T) {
	runner := &stubTrafficCollectRunner{
		machines: []model.Machine{
			{Base: model.Base{ID: 1}},
		},
		failures: map[uint]int{
			1: 1,
		},
	}

	scheduler := NewTrafficScheduler(config.CollectorConfig{
		Enabled:    true,
		Interval:   time.Hour,
		MaxWorkers: 1,
		RetryTimes: 1,
	}, runner, zerolog.Nop())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	scheduler.Start(ctx)
	time.Sleep(150 * time.Millisecond)
	scheduler.Stop()
	scheduler.Wait()

	value, ok := runner.calls.Load(uint(1))
	require.True(t, ok)
	require.Equal(t, 2, value.(int))
}

func TestTrafficSchedulerContinuesAfterStartContextCanceled(t *testing.T) {
	runner := &stubTrafficCollectRunner{
		machines: []model.Machine{
			{Base: model.Base{ID: 1}},
		},
		failures: map[uint]int{},
	}

	scheduler := NewTrafficScheduler(config.CollectorConfig{
		Enabled:    true,
		Interval:   40 * time.Millisecond,
		MaxWorkers: 1,
		RetryTimes: 0,
	}, runner, zerolog.Nop())

	startContext, cancel := context.WithCancel(context.Background())
	scheduler.Start(startContext)
	cancel()

	time.Sleep(120 * time.Millisecond)
	scheduler.Stop()
	scheduler.Wait()

	value, ok := runner.calls.Load(uint(1))
	require.True(t, ok)
	require.GreaterOrEqual(t, value.(int), 2)
}
