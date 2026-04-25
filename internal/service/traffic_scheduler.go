package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"traffic-monitor/internal/config"
	"traffic-monitor/internal/model"

	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

const trafficSchedulerRetryDelay = 100 * time.Millisecond

type TrafficCollectRunner interface {
	CollectMachine(ctx context.Context, machine *model.Machine) ([]model.TrafficSample, error)
	ListEnabledMachines(ctx context.Context) ([]model.Machine, error)
}

type TrafficScheduler struct {
	cfg         config.CollectorConfig
	collector   TrafficCollectRunner
	log         zerolog.Logger
	stopCh      chan struct{}
	stopOnce    sync.Once
	workersDone sync.WaitGroup
	loopDone    sync.WaitGroup
}

type trafficCollectJob struct {
	machine model.Machine
}

func NewTrafficCollectRunner(collector *TrafficCollectionService) TrafficCollectRunner {
	return collector
}

func NewTrafficScheduler(cfg config.CollectorConfig, collector TrafficCollectRunner, log zerolog.Logger) *TrafficScheduler {
	return &TrafficScheduler{
		cfg:       cfg,
		collector: collector,
		log:       log,
		stopCh:    make(chan struct{}),
	}
}

func RegisterTrafficScheduler(lifecycle fx.Lifecycle, scheduler *TrafficScheduler) {
	lifecycle.Append(fx.Hook{
		OnStart: func(startContext context.Context) error {
			if !scheduler.cfg.Enabled {
				scheduler.log.Info().Msg("traffic scheduler disabled")
				return nil
			}

			scheduler.Start(startContext)
			return nil
		},
		OnStop: func(stopContext context.Context) error {
			scheduler.Stop()
			done := make(chan struct{})
			go func() {
				scheduler.Wait()
				close(done)
			}()

			select {
			case <-stopContext.Done():
				return fmt.Errorf("wait scheduler stop: %w", stopContext.Err())
			case <-done:
				return nil
			}
		},
	})
}

func (scheduler *TrafficScheduler) Start(ctx context.Context) {
	jobs := make(chan trafficCollectJob, scheduler.cfg.MaxWorkers)
	scheduler.log.Info().
		Dur("interval", scheduler.cfg.Interval).
		Int("max_workers", scheduler.cfg.MaxWorkers).
		Int("retry_times", scheduler.cfg.RetryTimes).
		Msg("traffic scheduler started")

	for workerIndex := 0; workerIndex < scheduler.cfg.MaxWorkers; workerIndex++ {
		scheduler.workersDone.Add(1)
		go scheduler.runWorker(ctx, jobs)
	}

	scheduler.loopDone.Add(1)
	go scheduler.runLoop(ctx, jobs)
}

func (scheduler *TrafficScheduler) Stop() {
	scheduler.stopOnce.Do(func() {
		close(scheduler.stopCh)
	})
}

func (scheduler *TrafficScheduler) Wait() {
	scheduler.loopDone.Wait()
	scheduler.workersDone.Wait()
}

func (scheduler *TrafficScheduler) runLoop(ctx context.Context, jobs chan<- trafficCollectJob) {
	defer scheduler.loopDone.Done()
	defer close(jobs)

	ticker := time.NewTicker(scheduler.cfg.Interval)
	defer ticker.Stop()

	scheduler.dispatchOnce(ctx, jobs)

	for {
		select {
		case <-scheduler.stopCh:
			scheduler.log.Info().Msg("traffic scheduler stopped")
			return
		case <-ctx.Done():
			scheduler.log.Info().Msg("traffic scheduler context done")
			return
		case <-ticker.C:
			scheduler.dispatchOnce(ctx, jobs)
		}
	}
}

func (scheduler *TrafficScheduler) dispatchOnce(ctx context.Context, jobs chan<- trafficCollectJob) {
	machines, err := scheduler.collector.ListEnabledMachines(ctx)
	if err != nil {
		scheduler.log.Error().Err(err).Msg("list enabled machines failed")
		return
	}

	for _, machine := range machines {
		select {
		case <-scheduler.stopCh:
			return
		case <-ctx.Done():
			return
		case jobs <- trafficCollectJob{machine: machine}:
		}
	}
}

func (scheduler *TrafficScheduler) runWorker(ctx context.Context, jobs <-chan trafficCollectJob) {
	defer scheduler.workersDone.Done()

	for job := range jobs {
		scheduler.collectWithRetry(ctx, job.machine)
	}
}

func (scheduler *TrafficScheduler) collectWithRetry(ctx context.Context, machine model.Machine) {
	var lastErr error
	for attempt := 0; attempt <= scheduler.cfg.RetryTimes; attempt++ {
		startedAt := time.Now()
		samples, err := scheduler.collector.CollectMachine(ctx, &machine)
		if err == nil {
			scheduler.log.Info().
				Uint("machine_id", machine.ID).
				Str("host", machine.Host).
				Int("sample_count", len(samples)).
				Int("attempt", attempt+1).
				Dur("duration", time.Since(startedAt)).
				Msg("traffic collection succeeded")
			return
		}

		lastErr = err
		scheduler.log.Error().
			Err(err).
			Uint("machine_id", machine.ID).
			Str("host", machine.Host).
			Int("attempt", attempt+1).
			Dur("duration", time.Since(startedAt)).
			Msg("traffic collection failed")

		if attempt < scheduler.cfg.RetryTimes {
			select {
			case <-scheduler.stopCh:
				return
			case <-ctx.Done():
				return
			case <-time.After(trafficSchedulerRetryDelay):
			}
		}
	}

	if lastErr != nil && !errors.Is(lastErr, context.Canceled) {
		scheduler.log.Error().
			Err(lastErr).
			Uint("machine_id", machine.ID).
			Str("host", machine.Host).
			Msg("traffic collection exhausted retries")
	}
}
