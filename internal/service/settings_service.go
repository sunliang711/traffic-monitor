package service

import (
	"context"
	"fmt"
	"strconv"

	"traffic-monitor/internal/dto"
	"traffic-monitor/internal/model"
	"traffic-monitor/internal/repo"
)

const guestModeSettingKey = "guest_mode_enabled"

type SettingsStore interface {
	GetByKey(ctx context.Context, key string) (*model.AppSetting, error)
	Upsert(ctx context.Context, setting *model.AppSetting) error
}

type SettingsService struct {
	settingsStore SettingsStore
}

func NewSettingsService(settingsStore SettingsStore) *SettingsService {
	return &SettingsService{
		settingsStore: settingsStore,
	}
}

func (service *SettingsService) GetGuestMode(ctx context.Context) (dto.GuestModeResp, error) {
	enabled, err := service.IsGuestModeEnabled(ctx)
	if err != nil {
		return dto.GuestModeResp{}, err
	}

	return dto.GuestModeResp{Enabled: enabled}, nil
}

func (service *SettingsService) UpdateGuestMode(ctx context.Context, req dto.UpdateGuestModeReq) (dto.GuestModeResp, error) {
	setting := &model.AppSetting{
		SettingKey:   guestModeSettingKey,
		SettingValue: strconv.FormatBool(req.Enabled),
	}
	if err := service.settingsStore.Upsert(ctx, setting); err != nil {
		return dto.GuestModeResp{}, fmt.Errorf("update guest mode: %w", err)
	}

	return dto.GuestModeResp{Enabled: req.Enabled}, nil
}

func (service *SettingsService) IsGuestModeEnabled(ctx context.Context) (bool, error) {
	setting, err := service.settingsStore.GetByKey(ctx, guestModeSettingKey)
	if err != nil {
		if repo.IsRecordNotFound(err) {
			return false, nil
		}

		return false, fmt.Errorf("get guest mode setting: %w", err)
	}

	enabled, err := strconv.ParseBool(setting.SettingValue)
	if err != nil {
		return false, fmt.Errorf("parse guest mode setting: %w", err)
	}

	return enabled, nil
}
