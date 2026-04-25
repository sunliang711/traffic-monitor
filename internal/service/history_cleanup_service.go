package service

import (
	"context"
	"errors"
	"fmt"
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
	cfg    config.HistoryCleanupConfig
	runner HistoryCleanupRunner
	log    zerolog.Logger
	stopCh chan struct{}
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

func RegisterHistoryCleanupScheduler(lifecycle fx.Lifecycle, scheduler *HistoryCleanupScheduler) {
	lifecycle.Append(fx.Hook{
		OnStart: func(startContext context.Context) error {
			if !scheduler.cfg.Enabled {
				scheduler.log.Info().Msg("history cleanup scheduler disabled")
				return nil
			}

			scheduler.Start(startContext)
			return nil
		},
		OnStop: func(context.Context) error {
			scheduler.Stop()
			return nil
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
	scheduler.log.Info().
		Dur("interval", scheduler.cfg.Interval).
		Int("samples_days", scheduler.cfg.SamplesDays).
		Int("alerts_days", scheduler.cfg.AlertsDays).
		Int("batch_size", scheduler.cfg.BatchSize).
		Dur("timeout", scheduler.cfg.Timeout).
		Msg("history cleanup scheduler started")

	go scheduler.runLoop(ctx)
}

func (scheduler *HistoryCleanupScheduler) Stop() {
	select {
	case <-scheduler.stopCh:
		return
	default:
		close(scheduler.stopCh)
	}
}

func (scheduler *HistoryCleanupScheduler) runLoop(ctx context.Context) {
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
