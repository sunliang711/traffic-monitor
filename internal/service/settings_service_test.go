package service

import (
	"context"
	"testing"

	"traffic-monitor/internal/dto"
	"traffic-monitor/internal/model"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type stubSettingsStore struct {
	setting *model.AppSetting
}

func (store *stubSettingsStore) GetByKey(_ context.Context, _ string) (*model.AppSetting, error) {
	if store.setting == nil {
		return nil, gorm.ErrRecordNotFound
	}

	return store.setting, nil
}

func (store *stubSettingsStore) Upsert(_ context.Context, setting *model.AppSetting) error {
	store.setting = setting
	return nil
}

func TestSettingsServiceIsGuestModeEnabled_DefaultsToFalse(t *testing.T) {
	service := NewSettingsService(&stubSettingsStore{})

	enabled, err := service.IsGuestModeEnabled(context.Background())

	require.NoError(t, err)
	require.False(t, enabled)
}

func TestSettingsServiceUpdateGuestMode(t *testing.T) {
	store := &stubSettingsStore{}
	service := NewSettingsService(store)

	response, err := service.UpdateGuestMode(context.Background(), dto.UpdateGuestModeReq{Enabled: true})

	require.NoError(t, err)
	require.True(t, response.Enabled)
	require.Equal(t, "true", store.setting.SettingValue)
}
