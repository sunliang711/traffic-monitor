package service

import (
	"context"

	"traffic-monitor/internal/config"
	"traffic-monitor/internal/dto"
	"traffic-monitor/internal/repo"
)

type HealthService struct {
	appConfig  config.AppConfig
	healthRepo *repo.HealthRepo
}

func NewHealthService(appConfig config.AppConfig, healthRepo *repo.HealthRepo) *HealthService {
	return &HealthService{
		appConfig:  appConfig,
		healthRepo: healthRepo,
	}
}

func (service *HealthService) Check(ctx context.Context) (dto.HealthResp, error) {
	if err := service.healthRepo.Ping(ctx); err != nil {
		return dto.HealthResp{
			AppName: service.appConfig.Name,
			DB:      "down",
			Env:     service.appConfig.Env,
			Status:  "degraded",
		}, err
	}

	return dto.HealthResp{
		AppName: service.appConfig.Name,
		DB:      "up",
		Env:     service.appConfig.Env,
		Status:  "ok",
	}, nil
}
