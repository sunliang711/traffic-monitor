package service

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"traffic-monitor/internal/dto"
	"traffic-monitor/internal/model"
	"traffic-monitor/internal/repo"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

type stubAlertStore struct {
	alerts map[string]*model.Alert
	list   []model.Alert
}

func (store *stubAlertStore) CreateIfAbsent(_ context.Context, alert *model.Alert) (bool, error) {
	if store.alerts == nil {
		store.alerts = make(map[string]*model.Alert)
	}

	if _, ok := store.alerts[alert.AlertKey]; ok {
		return false, nil
	}

	alert.ID = uint(len(store.alerts) + 1)
	store.alerts[alert.AlertKey] = alert
	store.list = append(store.list, *alert)
	return true, nil
}

func (store *stubAlertStore) UpdateNotifyStatus(_ context.Context, alertID uint, notifyStatus string) error {
	for index := range store.list {
		if store.list[index].ID == alertID {
			store.list[index].NotifyStatus = notifyStatus
			break
		}
	}

	for _, alert := range store.alerts {
		if alert.ID == alertID {
			alert.NotifyStatus = notifyStatus
		}
	}

	return nil
}

func (store *stubAlertStore) List(_ context.Context, _ repo.AlertFilter) ([]model.Alert, int64, error) {
	return store.list, int64(len(store.list)), nil
}

type stubNotificationChannelStore struct {
	channels []model.NotificationChannel
}

func (store *stubNotificationChannelStore) Upsert(_ context.Context, channel *model.NotificationChannel) error {
	for index := range store.channels {
		if store.channels[index].ChannelType == channel.ChannelType {
			store.channels[index] = *channel
			return nil
		}
	}

	store.channels = append(store.channels, *channel)
	return nil
}

func (store *stubNotificationChannelStore) List(_ context.Context) ([]model.NotificationChannel, error) {
	return store.channels, nil
}

type stubNotificationDeliveryStore struct {
	deliveries []model.NotificationDelivery
}

func (store *stubNotificationDeliveryStore) Create(_ context.Context, delivery *model.NotificationDelivery) error {
	store.deliveries = append(store.deliveries, *delivery)
	return nil
}

type stubThresholdProvider struct {
	rules []dto.ThresholdRuleResp
}

func (provider *stubThresholdProvider) ListEffectiveMachineRules(_ context.Context, _ uint) ([]dto.ThresholdRuleResp, error) {
	return provider.rules, nil
}

type stubHTTPClient struct {
	statusCode int
	body       string
	err        error
}

func (client stubHTTPClient) Do(_ *http.Request) (*http.Response, error) {
	if client.err != nil {
		return nil, client.err
	}

	return &http.Response{
		StatusCode: client.statusCode,
		Body:       ioNopCloser{reader: strings.NewReader(client.body)},
	}, nil
}

type ioNopCloser struct {
	reader *strings.Reader
}

func (closer ioNopCloser) Read(p []byte) (int, error) {
	return closer.reader.Read(p)
}

func (closer ioNopCloser) Close() error {
	return nil
}

func TestAlertServiceEvaluateSamplesDeduplicates(t *testing.T) {
	alertStore := &stubAlertStore{}
	service := &AlertService{
		alertStore:                alertStore,
		notificationChannelStore:  &stubNotificationChannelStore{},
		notificationDeliveryStore: &stubNotificationDeliveryStore{},
		thresholdProvider: &stubThresholdProvider{
			rules: []dto.ThresholdRuleResp{
				{PeriodType: thresholdPeriodHourly, MetricType: thresholdMetricTotal, ThresholdMB: 100, Enabled: true},
			},
		},
		httpClient: stubHTTPClient{statusCode: http.StatusOK, body: "ok"},
		log:        zerolog.Nop(),
	}

	samples := []model.TrafficSample{
		{
			MachineID:  1,
			PeriodType: thresholdPeriodHourly,
			BucketTime: time.Date(2026, 4, 25, 13, 0, 0, 0, time.UTC),
			TotalMB:    120,
		},
	}

	require.NoError(t, service.EvaluateSamples(context.Background(), 1, samples))
	require.NoError(t, service.EvaluateSamples(context.Background(), 1, samples))
	require.Len(t, alertStore.list, 1)
	require.Equal(t, alertNotifyStatusSkipped, alertStore.list[0].NotifyStatus)
}

func TestAlertServiceSendsWebhook(t *testing.T) {
	alertStore := &stubAlertStore{}
	channelStore := &stubNotificationChannelStore{
		channels: []model.NotificationChannel{
			{
				ChannelType: channelTypeWebhook,
				Enabled:     true,
				ConfigJSON:  `{"url":"https://example.com/hook","headers":{"X-Test":"1"}}`,
			},
		},
	}
	deliveryStore := &stubNotificationDeliveryStore{}

	service := &AlertService{
		alertStore:                alertStore,
		notificationChannelStore:  channelStore,
		notificationDeliveryStore: deliveryStore,
		thresholdProvider: &stubThresholdProvider{
			rules: []dto.ThresholdRuleResp{
				{PeriodType: thresholdPeriodDaily, MetricType: thresholdMetricUpload, ThresholdMB: 1, Enabled: true},
			},
		},
		httpClient: stubHTTPClient{statusCode: http.StatusOK, body: "ok"},
		log:        zerolog.Nop(),
	}

	samples := []model.TrafficSample{
		{
			MachineID:  2,
			PeriodType: thresholdPeriodDaily,
			BucketTime: time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC),
			UploadMB:   5,
		},
	}

	require.NoError(t, service.EvaluateSamples(context.Background(), 2, samples))
	require.Len(t, deliveryStore.deliveries, 1)
	require.True(t, deliveryStore.deliveries[0].Success)
}

func TestAlertServiceFailedNotification(t *testing.T) {
	alertStore := &stubAlertStore{}
	channelStore := &stubNotificationChannelStore{
		channels: []model.NotificationChannel{
			{
				ChannelType: channelTypeWebhook,
				Enabled:     true,
				ConfigJSON:  `{"url":"https://example.com/hook"}`,
			},
		},
	}

	service := &AlertService{
		alertStore:                alertStore,
		notificationChannelStore:  channelStore,
		notificationDeliveryStore: &stubNotificationDeliveryStore{},
		thresholdProvider: &stubThresholdProvider{
			rules: []dto.ThresholdRuleResp{
				{PeriodType: thresholdPeriodHourly, MetricType: thresholdMetricDownload, ThresholdMB: 10, Enabled: true},
			},
		},
		httpClient: stubHTTPClient{err: errors.New("network failed")},
		log:        zerolog.Nop(),
	}

	samples := []model.TrafficSample{
		{
			MachineID:  3,
			PeriodType: thresholdPeriodHourly,
			BucketTime: time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC),
			DownloadMB: 50,
		},
	}

	require.NoError(t, service.EvaluateSamples(context.Background(), 3, samples))
	require.Equal(t, alertNotifyStatusFailed, alertStore.list[0].NotifyStatus)
}
