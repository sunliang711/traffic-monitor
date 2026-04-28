package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"traffic-monitor/internal/config"
	"traffic-monitor/internal/dto"
	"traffic-monitor/internal/repo"

	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

var ErrInvalidHistoryCleanupRequest = errors.New("invalid history cleanup request")

type HistoryCleanupResult struct {
	DeletedSamples int64
	DeletedAlerts  int64
	SamplesCutoff  time.Time
	AlertsCutoff   time.Time
}

type HistoryCleanupService struct {
	cfg               config.HistoryCleanupConfig
	trafficSampleRepo *repo.TrafficSampleRepo
	alertRepo         *repo.AlertRepo
	log               zerolog.Logger
}

type HistoryCleanupRunner interface {
	RunOnce(ctx context.Context) (HistoryCleanupResult, error)
}

type HistoryCleanupScheduler struct {
	cfg         config.HistoryCleanupConfig
	runner      HistoryCleanupRunner
	log         zerolog.Logger
	stopCh      chan struct{}
	stopOnce    sync.Once
	runCancel   context.CancelFunc
	runCancelMu sync.Mutex
	loopDone    sync.WaitGroup
}

func NewHistoryCleanupService(
	cfg config.HistoryCleanupConfig,
	trafficSampleRepo *repo.TrafficSampleRepo,
	alertRepo *repo.AlertRepo,
	log zerolog.Logger,
) *HistoryCleanupService {
	return &HistoryCleanupService{
		cfg:               cfg,
		trafficSampleRepo: trafficSampleRepo,
		alertRepo:         alertRepo,
		log:               log.With().Str("component", "history_cleanup").Logger(),
	}
}

func NewHistoryCleanupRunner(service *HistoryCleanupService) HistoryCleanupRunner {
	return service
}

func NewHistoryCleanupScheduler(
	cfg config.HistoryCleanupConfig,
	runner HistoryCleanupRunner,
	log zerolog.Logger,
) *HistoryCleanupScheduler {
	return &HistoryCleanupScheduler{
		cfg:    cfg,
		runner: runner,
		log:    log.With().Str("component", "history_cleanup_scheduler").Logger(),
		stopCh: make(chan struct{}),
	}
}

func RegisterHistoryCleanupScheduler(lifecycle fx.Lifecycle, scheduler *HistoryCleanupScheduler, restoreConfig config.RestoreConfig) {
	lifecycle.Append(fx.Hook{
		OnStart: func(startContext context.Context) error {
			if restoreConfig.Enabled() {
				scheduler.log.Info().Msg("history cleanup scheduler disabled in restore mode")
				return nil
			}

			if !scheduler.cfg.Enabled {
				scheduler.log.Info().Msg("history cleanup scheduler disabled")
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
				return fmt.Errorf("wait history cleanup scheduler stop: %w", stopContext.Err())
			case <-done:
				return nil
			}
		},
	})
}

func (service *HistoryCleanupService) RunOnce(ctx context.Context) (HistoryCleanupResult, error) {
	req := dto.CleanupHistoryReq{
		DeleteSamples: true,
		DeleteAlerts:  true,
	}

	resp, err := service.Cleanup(ctx, req)
	if err != nil {
		return HistoryCleanupResult{}, err
	}

	return toHistoryCleanupResult(resp), nil
}

func (service *HistoryCleanupService) Cleanup(ctx context.Context, req dto.CleanupHistoryReq) (dto.CleanupHistoryResp, error) {
	if !req.DeleteSamples && !req.DeleteAlerts {
		req.DeleteSamples = true
		req.DeleteAlerts = true
	}

	samplesDays := service.cfg.SamplesDays
	if req.SamplesDays != nil {
		samplesDays = *req.SamplesDays
	}

	alertsDays := service.cfg.AlertsDays
	if req.AlertsDays != nil {
		alertsDays = *req.AlertsDays
	}

	if req.DeleteSamples && samplesDays <= 0 {
		return dto.CleanupHistoryResp{}, fmt.Errorf("%w: samples_days must be greater than zero", ErrInvalidHistoryCleanupRequest)
	}

	if req.DeleteAlerts && alertsDays <= 0 {
		return dto.CleanupHistoryResp{}, fmt.Errorf("%w: alerts_days must be greater than zero", ErrInvalidHistoryCleanupRequest)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, service.cfg.Timeout)
	defer cancel()

	now := time.Now().UTC()
	resp := dto.CleanupHistoryResp{}

	if req.DeleteSamples {
		resp.SamplesCutoff = now.AddDate(0, 0, -samplesDays)
		deletedSamples, err := service.deleteExpiredSamples(timeoutCtx, req.MachineID, resp.SamplesCutoff)
		if err != nil {
			return dto.CleanupHistoryResp{}, err
		}
		resp.DeletedSamples = deletedSamples
	}

	if req.DeleteAlerts {
		resp.AlertsCutoff = now.AddDate(0, 0, -alertsDays)
		deletedAlerts, err := service.deleteExpiredAlerts(timeoutCtx, req.MachineID, resp.AlertsCutoff)
		if err != nil {
			return dto.CleanupHistoryResp{}, err
		}
		resp.DeletedAlerts = deletedAlerts
	}

	service.log.Info().
		Bool("delete_samples", req.DeleteSamples).
		Bool("delete_alerts", req.DeleteAlerts).
		Int64("deleted_samples", resp.DeletedSamples).
		Int64("deleted_alerts", resp.DeletedAlerts).
		Time("samples_cutoff", resp.SamplesCutoff).
		Time("alerts_cutoff", resp.AlertsCutoff).
		Msg("history cleanup completed")

	return resp, nil
}

func (scheduler *HistoryCleanupScheduler) Start(ctx context.Context) {
	runContext, runCancel := context.WithCancel(context.Background())
	scheduler.setRunCancel(runCancel)

	// fx 的 startContext 只覆盖启动阶段，后台清理要使用独立运行上下文。
	scheduler.log.Info().
		Dur("interval", scheduler.cfg.Interval).
		Int("samples_days", scheduler.cfg.SamplesDays).
		Int("alerts_days", scheduler.cfg.AlertsDays).
		Int("batch_size", scheduler.cfg.BatchSize).
		Dur("timeout", scheduler.cfg.Timeout).
		Msg("history cleanup scheduler started")

	scheduler.loopDone.Add(1)
	go scheduler.runLoop(runContext)
}

func (scheduler *HistoryCleanupScheduler) Stop() {
	scheduler.stopOnce.Do(func() {
		scheduler.cancelRun()
		close(scheduler.stopCh)
	})
}

func (scheduler *HistoryCleanupScheduler) Wait() {
	scheduler.loopDone.Wait()
}

func (scheduler *HistoryCleanupScheduler) runLoop(ctx context.Context) {
	defer scheduler.loopDone.Done()

	ticker := time.NewTicker(scheduler.cfg.Interval)
	defer ticker.Stop()

	scheduler.runOnce(ctx)

	for {
		select {
		case <-scheduler.stopCh:
			scheduler.log.Info().Msg("history cleanup scheduler stopped")
			return
		case <-ctx.Done():
			scheduler.log.Info().Msg("history cleanup scheduler context done")
			return
		case <-ticker.C:
			scheduler.runOnce(ctx)
		}
	}
}

func (scheduler *HistoryCleanupScheduler) runOnce(ctx context.Context) {
	result, err := scheduler.runner.RunOnce(ctx)
	if err != nil {
		scheduler.log.Error().Err(err).Msg("history cleanup run failed")
		return
	}

	scheduler.log.Info().
		Int64("deleted_samples", result.DeletedSamples).
		Int64("deleted_alerts", result.DeletedAlerts).
		Time("samples_cutoff", result.SamplesCutoff).
		Time("alerts_cutoff", result.AlertsCutoff).
		Msg("history cleanup run completed")
}

func (scheduler *HistoryCleanupScheduler) setRunCancel(runCancel context.CancelFunc) {
	scheduler.runCancelMu.Lock()
	defer scheduler.runCancelMu.Unlock()

	scheduler.runCancel = runCancel
}

func (scheduler *HistoryCleanupScheduler) cancelRun() {
	scheduler.runCancelMu.Lock()
	defer scheduler.runCancelMu.Unlock()

	if scheduler.runCancel != nil {
		scheduler.runCancel()
		scheduler.runCancel = nil
	}
}

func (service *HistoryCleanupService) deleteExpiredSamples(ctx context.Context, machineID *uint, cutoff time.Time) (int64, error) {
	var totalDeleted int64

	for {
		beforeBucketTime := cutoff
		deleted, err := service.trafficSampleRepo.DeleteBefore(ctx, repo.TrafficSampleDeleteFilter{
			MachineID:        machineID,
			BeforeBucketTime: &beforeBucketTime,
			Limit:            service.cfg.BatchSize,
		})
		if err != nil {
			return 0, fmt.Errorf("delete expired traffic samples: %w", err)
		}

		totalDeleted += deleted
		if deleted < int64(service.cfg.BatchSize) {
			return totalDeleted, nil
		}
	}
}

func (service *HistoryCleanupService) deleteExpiredAlerts(ctx context.Context, machineID *uint, cutoff time.Time) (int64, error) {
	var totalDeleted int64

	for {
		deleted, err := service.alertRepo.DeleteExpired(ctx, repo.AlertDeleteFilter{
			MachineID:        machineID,
			BeforeBucketTime: cutoff,
			Limit:            service.cfg.BatchSize,
		})
		if err != nil {
			return 0, fmt.Errorf("delete expired alerts: %w", err)
		}

		totalDeleted += deleted
		if deleted < int64(service.cfg.BatchSize) {
			return totalDeleted, nil
		}
	}
}

func toHistoryCleanupResult(resp dto.CleanupHistoryResp) HistoryCleanupResult {
	return HistoryCleanupResult{
		DeletedSamples: resp.DeletedSamples,
		DeletedAlerts:  resp.DeletedAlerts,
		SamplesCutoff:  resp.SamplesCutoff,
		AlertsCutoff:   resp.AlertsCutoff,
	}
}

func (service *HistoryCleanupService) CleanupResult(ctx context.Context, req dto.CleanupHistoryReq) (HistoryCleanupResult, error) {
	resp, err := service.Cleanup(ctx, req)
	if err != nil {
		return HistoryCleanupResult{}, err
	}
	return toHistoryCleanupResult(resp), nil
}
