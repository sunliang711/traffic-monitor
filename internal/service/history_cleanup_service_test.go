package service

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"traffic-monitor/internal/config"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

type stubHistoryCleanupRunner struct {
	runCount atomic.Int32
}

func (runner *stubHistoryCleanupRunner) RunOnce(_ context.Context) (HistoryCleanupResult, error) {
	runner.runCount.Add(1)
	return HistoryCleanupResult{}, nil
}

func TestHistoryCleanupSchedulerContinuesAfterStartContextCanceled(t *testing.T) {
	runner := &stubHistoryCleanupRunner{}
	scheduler := NewHistoryCleanupScheduler(config.HistoryCleanupConfig{
		Enabled:     true,
		Interval:    40 * time.Millisecond,
		SamplesDays: 3,
		AlertsDays:  3,
		BatchSize:   100,
		Timeout:     5 * time.Second,
	}, runner, zerolog.Nop())

	startContext, cancel := context.WithCancel(context.Background())
	scheduler.Start(startContext)
	cancel()

	time.Sleep(120 * time.Millisecond)
	scheduler.Stop()
	scheduler.Wait()

	require.GreaterOrEqual(t, runner.runCount.Load(), int32(2))
}
