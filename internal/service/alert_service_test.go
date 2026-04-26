package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
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

type stubNotificationProxyStore struct {
	proxies []model.NotificationProxy
}

func (store *stubNotificationProxyStore) Create(_ context.Context, notificationProxy *model.NotificationProxy) error {
	if notificationProxy.ID == 0 {
		notificationProxy.ID = uint(len(store.proxies) + 1)
	}
	store.proxies = append(store.proxies, *notificationProxy)
	return nil
}

func (store *stubNotificationProxyStore) Update(_ context.Context, notificationProxy *model.NotificationProxy) error {
	for index := range store.proxies {
		if store.proxies[index].ID == notificationProxy.ID {
			store.proxies[index] = *notificationProxy
			return nil
		}
	}

	return errors.New("notification proxy not found")
}

func (store *stubNotificationProxyStore) Delete(_ context.Context, proxyID uint) error {
	for index := range store.proxies {
		if store.proxies[index].ID == proxyID {
			store.proxies = append(store.proxies[:index], store.proxies[index+1:]...)
			return nil
		}
	}

	return errors.New("notification proxy not found")
}

func (store *stubNotificationProxyStore) GetByID(_ context.Context, proxyID uint) (*model.NotificationProxy, error) {
	for index := range store.proxies {
		if store.proxies[index].ID == proxyID {
			return &store.proxies[index], nil
		}
	}

	return nil, errors.New("notification proxy not found")
}

func (store *stubNotificationProxyStore) List(_ context.Context) ([]model.NotificationProxy, error) {
	return store.proxies, nil
}

type stubNotificationDeliveryStore struct {
	deliveries []model.NotificationDelivery
}

func (store *stubNotificationDeliveryStore) Create(_ context.Context, delivery *model.NotificationDelivery) error {
	store.deliveries = append(store.deliveries, *delivery)
	return nil
}

func (store *stubNotificationDeliveryStore) GetLatestByAlertIDs(_ context.Context, alertIDs []uint) (map[uint]model.NotificationDelivery, error) {
	result := make(map[uint]model.NotificationDelivery)
	if len(alertIDs) == 0 {
		return result, nil
	}

	alertIDSet := make(map[uint]struct{}, len(alertIDs))
	for _, alertID := range alertIDs {
		alertIDSet[alertID] = struct{}{}
	}

	for index := len(store.deliveries) - 1; index >= 0; index-- {
		delivery := store.deliveries[index]
		if _, ok := alertIDSet[delivery.AlertID]; !ok {
			continue
		}
		if _, exists := result[delivery.AlertID]; exists {
			continue
		}
		result[delivery.AlertID] = delivery
	}

	return result, nil
}

type stubThresholdProvider struct {
	rules []dto.ThresholdRuleResp
}

func (provider *stubThresholdProvider) ListEffectiveMachineRules(_ context.Context, _ uint) ([]dto.ThresholdRuleResp, error) {
	return provider.rules, nil
}

type stubAlertMachineStore struct {
	machines map[uint]model.Machine
}

func (store *stubAlertMachineStore) GetByID(_ context.Context, machineID uint) (*model.Machine, error) {
	machine, ok := store.machines[machineID]
	if !ok {
		return nil, errors.New("machine not found")
	}

	return &machine, nil
}

type stubHTTPClient struct {
	statusCode int
	body       string
	err        error
	doFunc     func(req *http.Request) (*http.Response, error)
	requests   []*http.Request
}

func cloneRequest(req *http.Request) *http.Request {
	cloned := req.Clone(req.Context())
	if req.Body == nil {
		return cloned
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return cloned
	}

	_ = req.Body.Close()
	req.Body = ioNopCloser{reader: strings.NewReader(string(body))}
	cloned.Body = ioNopCloser{reader: strings.NewReader(string(body))}
	return cloned
}

func (client *stubHTTPClient) Do(req *http.Request) (*http.Response, error) {
	client.requests = append(client.requests, cloneRequest(req))

	if client.doFunc != nil {
		return client.doFunc(req)
	}

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

func uintPtr(value uint) *uint {
	return &value
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
		httpClient: &stubHTTPClient{statusCode: http.StatusOK, body: "ok"},
		log:        zerolog.Nop(),
	}

	samples := []model.TrafficSample{
		{
			MachineID:  1,
			PeriodType: thresholdPeriodHourly,
			BucketTime: time.Now().UTC().Truncate(time.Hour).Add(-time.Hour),
			TotalMB:    120,
		},
	}

	require.NoError(t, service.EvaluateSamples(context.Background(), 1, samples))
	require.NoError(t, service.EvaluateSamples(context.Background(), 1, samples))
	require.Len(t, alertStore.list, 1)
	require.Equal(t, alertNotifyStatusSkipped, alertStore.list[0].NotifyStatus)
}

func TestAlertServiceSkipsCurrentIncompleteHourlyPeriod(t *testing.T) {
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
		httpClient: &stubHTTPClient{statusCode: http.StatusOK, body: "ok"},
		log:        zerolog.Nop(),
	}

	samples := []model.TrafficSample{
		{
			MachineID:    1,
			PeriodType:   thresholdPeriodHourly,
			BucketTime:   time.Now().UTC().Truncate(time.Hour),
			TotalMB:      120,
			CollectedAt:  time.Now().UTC(),
		},
	}

	require.NoError(t, service.EvaluateSamples(context.Background(), 1, samples))
	require.Empty(t, alertStore.list)
}

func TestAlertServiceSkipsCurrentIncompleteDailyPeriod(t *testing.T) {
	alertStore := &stubAlertStore{}
	service := &AlertService{
		alertStore:                alertStore,
		notificationChannelStore:  &stubNotificationChannelStore{},
		notificationDeliveryStore: &stubNotificationDeliveryStore{},
		thresholdProvider: &stubThresholdProvider{
			rules: []dto.ThresholdRuleResp{
				{PeriodType: thresholdPeriodDaily, MetricType: thresholdMetricTotal, ThresholdMB: 100, Enabled: true},
			},
		},
		httpClient: &stubHTTPClient{statusCode: http.StatusOK, body: "ok"},
		log:        zerolog.Nop(),
	}

	now := time.Now().UTC()
	samples := []model.TrafficSample{
		{
			MachineID:    1,
			PeriodType:   thresholdPeriodDaily,
			BucketTime:   time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC),
			TotalMB:      120,
			CollectedAt:  now,
		},
	}

	require.NoError(t, service.EvaluateSamples(context.Background(), 1, samples))
	require.Empty(t, alertStore.list)
}

func TestAlertServiceSendsWebhookPOST(t *testing.T) {
	alertStore := &stubAlertStore{}
	channelStore := &stubNotificationChannelStore{
		channels: []model.NotificationChannel{
			{
				ChannelType: channelTypeWebhook,
				Enabled:     true,
				ConfigJSON:  `{"url":"https://example.com/hook?machine={{.machine_id}}&metric={{.metric_type}}","method":"POST","headers":{"X-Test":"{{.machine_id}}","X-Alert-Key":"{{.alert_key}}"},"body":"{\"machine_id\":{{.machine_id}},\"metric_type\":\"{{.metric_type}}\",\"actual_mb\":{{.actual_mb}},\"threshold_mb\":{{.threshold_mb}},\"bucket_time\":\"{{.bucket_time_rfc3339}}\"}"}`,
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
		httpClient: &stubHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				require.Equal(t, http.MethodPost, req.Method)
				require.Equal(t, "2", req.Header.Get("X-Test"))
				require.Regexp(t, `^2:daily:upload:\d+$`, req.Header.Get("X-Alert-Key"))
				require.Equal(t, "application/json", req.Header.Get("Content-Type"))
				require.Equal(t, "https://example.com/hook?machine=2&metric=upload", req.URL.String())

				body, err := io.ReadAll(req.Body)
				require.NoError(t, err)
				require.Contains(t, string(body), `"machine_id":2`)
				require.Contains(t, string(body), `"metric_type":"upload"`)
				require.Contains(t, string(body), `"actual_mb":5`)
				require.Contains(t, string(body), `"threshold_mb":1`)
				require.Regexp(t, `"bucket_time":"\d{4}-\d{2}-\d{2}T00:00:00Z"`, string(body))

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       ioNopCloser{reader: strings.NewReader("ok")},
				}, nil
			},
		},
		log: zerolog.Nop(),
	}

	now := time.Now().UTC()
	samples := []model.TrafficSample{
		{
			MachineID:  2,
			PeriodType: thresholdPeriodDaily,
			BucketTime: time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1),
			UploadMB:   5,
		},
	}

	require.NoError(t, service.EvaluateSamples(context.Background(), 2, samples))
	require.Len(t, deliveryStore.deliveries, 1)
	require.True(t, deliveryStore.deliveries[0].Success)
}

func TestAlertServiceSendsWebhookGET(t *testing.T) {
	alertStore := &stubAlertStore{}
	channelStore := &stubNotificationChannelStore{
		channels: []model.NotificationChannel{
			{
				ChannelType: channelTypeWebhook,
				Enabled:     true,
				ConfigJSON:  `{"url":"https://example.com/hook?machine={{.machine_id}}&metric={{.metric_type}}&bucket={{.bucket_time_rfc3339}}","method":"GET","headers":{"X-Test":"{{.machine_id}}","X-Alert-Key":"{{.alert_key}}"}}`,
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
		httpClient: &stubHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				require.Equal(t, http.MethodGet, req.Method)
				require.Equal(t, "2", req.Header.Get("X-Test"))
				require.Regexp(t, `^2:daily:upload:\d+$`, req.Header.Get("X-Alert-Key"))
				require.Empty(t, req.Header.Get("Content-Type"))

				parsedURL, err := url.Parse(req.URL.String())
				require.NoError(t, err)
				require.Equal(t, "2", parsedURL.Query().Get("machine"))
				require.Equal(t, "upload", parsedURL.Query().Get("metric"))
				require.Regexp(t, `^\d{4}-\d{2}-\d{2}T00:00:00Z$`, parsedURL.Query().Get("bucket"))

				if req.Body != nil {
					body, err := io.ReadAll(req.Body)
					require.NoError(t, err)
					require.Empty(t, string(body))
				}

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       ioNopCloser{reader: strings.NewReader("ok")},
				}, nil
			},
		},
		log: zerolog.Nop(),
	}

	now := time.Now().UTC()
	samples := []model.TrafficSample{
		{
			MachineID:  2,
			PeriodType: thresholdPeriodDaily,
			BucketTime: time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1),
			UploadMB:   5,
		},
	}

	require.NoError(t, service.EvaluateSamples(context.Background(), 2, samples))
	require.Len(t, deliveryStore.deliveries, 1)
	require.True(t, deliveryStore.deliveries[0].Success)
}

func TestAlertServiceSupportsBareWebhookTemplateVariables(t *testing.T) {
	alertStore := &stubAlertStore{}
	channelStore := &stubNotificationChannelStore{
		channels: []model.NotificationChannel{
			{
				ChannelType: channelTypeWebhook,
				Enabled:     true,
				ConfigJSON:  `{"url":"https://example.com/hook?machine={{machine_id}}&metric={{metric_type}}","method":"POST","headers":{"X-Test":"{{machine_id}}","X-Alert-Key":"{{alert_key}}"},"body":"{\"machine_id\":{{machine_id}},\"metric_type\":\"{{metric_type}}\",\"actual_mb\":{{actual_mb}},\"threshold_mb\":{{threshold_mb}},\"bucket_time\":\"{{bucket_time_rfc3339}}\"}"}`,
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
		httpClient: &stubHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				require.Equal(t, http.MethodPost, req.Method)
				require.Equal(t, "2", req.Header.Get("X-Test"))
				require.Regexp(t, `^2:daily:upload:\d+$`, req.Header.Get("X-Alert-Key"))
				require.Equal(t, "application/json", req.Header.Get("Content-Type"))
				require.Equal(t, "https://example.com/hook?machine=2&metric=upload", req.URL.String())

				body, err := io.ReadAll(req.Body)
				require.NoError(t, err)
				require.Contains(t, string(body), `"machine_id":2`)
				require.Contains(t, string(body), `"metric_type":"upload"`)
				require.Contains(t, string(body), `"actual_mb":5`)
				require.Contains(t, string(body), `"threshold_mb":1`)
				require.Regexp(t, `"bucket_time":"\d{4}-\d{2}-\d{2}T00:00:00Z"`, string(body))

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       ioNopCloser{reader: strings.NewReader("ok")},
				}, nil
			},
		},
		log: zerolog.Nop(),
	}

	now := time.Now().UTC()
	samples := []model.TrafficSample{
		{
			MachineID:  2,
			PeriodType: thresholdPeriodDaily,
			BucketTime: time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1),
			UploadMB:   5,
		},
	}

	require.NoError(t, service.EvaluateSamples(context.Background(), 2, samples))
	require.Len(t, deliveryStore.deliveries, 1)
	require.True(t, deliveryStore.deliveries[0].Success)
}

func TestAlertServiceAggregatesTelegramAlerts(t *testing.T) {
	alertStore := &stubAlertStore{}
	channelStore := &stubNotificationChannelStore{
		channels: []model.NotificationChannel{
			{
				ChannelType: channelTypeTelegram,
				Enabled:     true,
				ConfigJSON:  `{"bot_token":"1234567890abcdef","chat_id":"10001","message":"{{metric_type}}={{actual_human_readable}}/{{threshold_human_readable}}"}`,
			},
		},
	}
	deliveryStore := &stubNotificationDeliveryStore{}
	httpClient := &stubHTTPClient{
		statusCode: http.StatusOK,
		body:       `{"ok":true}`,
	}
	service := &AlertService{
		alertStore:                alertStore,
		notificationChannelStore:  channelStore,
		notificationDeliveryStore: deliveryStore,
		thresholdProvider: &stubThresholdProvider{
			rules: []dto.ThresholdRuleResp{
				{PeriodType: thresholdPeriodDaily, MetricType: thresholdMetricUpload, ThresholdMB: 100, Enabled: true},
				{PeriodType: thresholdPeriodDaily, MetricType: thresholdMetricDownload, ThresholdMB: 100, Enabled: true},
			},
		},
		httpClient: httpClient,
		log:        zerolog.Nop(),
	}

	samples := []model.TrafficSample{
		{
			MachineID:   2,
			PeriodType:  thresholdPeriodDaily,
			BucketTime:  time.Now().UTC().AddDate(0, 0, -1),
			UploadMB:    120,
			DownloadMB:  130,
			CollectedAt: time.Now().UTC(),
		},
	}

	require.NoError(t, service.EvaluateSamples(context.Background(), 2, samples))
	require.Len(t, alertStore.list, 2)
	require.Len(t, deliveryStore.deliveries, 2)
	require.Len(t, httpClient.requests, 1)
	require.Equal(t, alertNotifyStatusSent, alertStore.list[0].NotifyStatus)
	require.Equal(t, alertNotifyStatusSent, alertStore.list[1].NotifyStatus)

	req := httpClient.requests[0]
	require.Equal(t, http.MethodPost, req.Method)
	require.Equal(t, "https://api.telegram.org/bot1234567890abcdef/sendMessage", req.URL.String())

	body, err := io.ReadAll(req.Body)
	require.NoError(t, err)

	var payload map[string]string
	require.NoError(t, json.Unmarshal(body, &payload))
	require.Equal(t, "10001", payload["chat_id"])
	require.Equal(t, "upload=120.000 MB/100.000 MB\n\ndownload=130.000 MB/100.000 MB", payload["text"])
}

func TestAlertServiceListNotificationChannelsIncludesWebhookTemplates(t *testing.T) {
	service := &AlertService{
		notificationChannelStore: &stubNotificationChannelStore{
			channels: []model.NotificationChannel{
				{
					ChannelType: channelTypeWebhook,
					Enabled:     true,
					ConfigJSON:  `{"url":"https://example.com/hook?machine={{.machine_id}}","method":"POST","headers":{"X-Test":"{{.machine_id}}"},"body":"{\"machine_id\":\"{{.machine_id}}\"}","proxy_id":3}`,
				},
				{
					ChannelType: channelTypeTelegram,
					Enabled:     true,
					ConfigJSON:  `{"bot_token":"1234567890abcdef","chat_id":"10001","message":"{{machine_name}} {{actual_human_readable}}","proxy_id":4}`,
				},
			},
		},
		log: zerolog.Nop(),
	}

	channels, err := service.ListNotificationChannels(context.Background())
	require.NoError(t, err)
	require.Len(t, channels, 2)

	webhook := channels[0]
	require.Equal(t, channelTypeWebhook, webhook.ChannelType)
	require.True(t, webhook.Enabled)
	require.True(t, webhook.Configured)
	require.Equal(t, http.MethodPost, webhook.Method)
	require.Equal(t, "https://example.com/hook?machine={{.machine_id}}", webhook.URL)
	require.Equal(t, map[string]string{"X-Test": "{{.machine_id}}"}, webhook.Headers)
	require.Equal(t, "{\"machine_id\":\"{{.machine_id}}\"}", webhook.Body)
	require.Equal(t, uint(3), *webhook.ProxyID)

	telegram := channels[1]
	require.Equal(t, channelTypeTelegram, telegram.ChannelType)
	require.Equal(t, "10001", telegram.ChatID)
	require.Equal(t, "{{machine_name}} {{actual_human_readable}}", telegram.Message)
	require.Equal(t, "1234...cdef", telegram.TokenMasked)
	require.Equal(t, uint(4), *telegram.ProxyID)
}

func TestAlertServiceBuildWebhookTemplateDataIncludesMachineFields(t *testing.T) {
	service := &AlertService{
		machineStore: &stubAlertMachineStore{
			machines: map[uint]model.Machine{
				7: {
					Name: "prod-api-01",
					Host: "10.2.1.107",
				},
			},
		},
		log: zerolog.Nop(),
	}

	alert := &model.Alert{
		MachineID:   7,
		PeriodType:  thresholdPeriodDaily,
		MetricType:  thresholdMetricTotal,
		BucketTime:  time.Date(2026, 4, 25, 8, 0, 0, 0, time.UTC),
		ThresholdMB: 1024,
		ActualMB:    1536,
		AlertKey:    "7:daily:total:1777104000",
	}

	data := service.buildWebhookTemplateData(context.Background(), alert)
	require.Equal(t, uint(7), data["machine_id"])
	require.Equal(t, "prod-api-01", data["machine_name"])
	require.Equal(t, "10.2.1.107", data["machine_host"])
	require.Equal(t, thresholdPeriodDaily, data["period_type"])
	require.Equal(t, thresholdMetricTotal, data["metric_type"])
	require.Equal(t, "7:daily:total:1777104000", data["alert_key"])
}

func TestAlertServiceRenderWebhookRequestIncludesMachineFields(t *testing.T) {
	service := &AlertService{
		machineStore: &stubAlertMachineStore{
			machines: map[uint]model.Machine{
				7: {
					Name: "prod-api-01",
					Host: "10.2.1.107",
				},
			},
		},
		log: zerolog.Nop(),
	}

	alert := &model.Alert{
		MachineID:   7,
		PeriodType:  thresholdPeriodDaily,
		MetricType:  thresholdMetricTotal,
		BucketTime:  time.Date(2026, 4, 25, 8, 0, 0, 0, time.UTC),
		ThresholdMB: 1024,
		ActualMB:    1536,
		AlertKey:    "7:daily:total:1777104000",
	}

	req, previewJSON, err := service.renderWebhookRequest(context.Background(), `{"url":"https://example.com/hook?machine={{machine_name}}&host={{machine_host}}","method":"POST","headers":{"X-Machine-Name":"{{machine_name}}","X-Machine-Host":"{{machine_host}}"},"body":"{\"machine\":\"{{machine_name}}\",\"host\":\"{{machine_host}}\",\"metric\":\"{{metric_type}}\"}"}`, alert)
	require.NoError(t, err)
	require.Equal(t, http.MethodPost, req.Method)
	require.Equal(t, "https://example.com/hook?machine=prod-api-01&host=10.2.1.107", req.URL.String())
	require.Equal(t, "prod-api-01", req.Header.Get("X-Machine-Name"))
	require.Equal(t, "10.2.1.107", req.Header.Get("X-Machine-Host"))

	body, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"machine":"prod-api-01","host":"10.2.1.107","metric":"total"}`, string(body))

	var preview struct {
		Method  string            `json:"method"`
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
		Body    string            `json:"body"`
	}
	require.NoError(t, json.Unmarshal([]byte(previewJSON), &preview))
	require.Equal(t, http.MethodPost, preview.Method)
	require.Equal(t, "https://example.com/hook?machine=prod-api-01&host=10.2.1.107", preview.URL)
	require.Equal(t, map[string]string{
		"X-Machine-Name": "prod-api-01",
		"X-Machine-Host": "10.2.1.107",
	}, preview.Headers)
	require.JSONEq(t, `{"machine":"prod-api-01","host":"10.2.1.107","metric":"total"}`, preview.Body)
}

func TestAlertServiceUpsertWebhookChannelPersistsTemplates(t *testing.T) {
	channelStore := &stubNotificationChannelStore{}
	service := &AlertService{
		notificationChannelStore: channelStore,
		notificationProxyStore: &stubNotificationProxyStore{
			proxies: []model.NotificationProxy{
				{Base: model.Base{ID: 9}, Name: "proxy-9", ProxyType: "http", URL: "http://127.0.0.1:7890"},
			},
		},
		log: zerolog.Nop(),
	}

	err := service.UpsertWebhookChannel(context.Background(), dto.UpsertWebhookChannelReq{
		Enabled: true,
		Method:  "post",
		URL:     "https://example.com/hook?machine={{.machine_id}}",
		Headers: map[string]string{
			"X-Test": "{{.machine_id}}",
		},
		Body:    "{\"machine_id\":\"{{.machine_id}}\",\"metric\":\"{{.metric_type}}\"}",
		ProxyID: uintPtr(9),
	})
	require.NoError(t, err)
	require.Len(t, channelStore.channels, 1)

	channel := channelStore.channels[0]
	require.Equal(t, channelTypeWebhook, channel.ChannelType)
	require.True(t, channel.Enabled)

	var cfg struct {
		URL     string            `json:"url"`
		Method  string            `json:"method"`
		Headers map[string]string `json:"headers"`
		Body    string            `json:"body"`
		ProxyID *uint             `json:"proxy_id"`
	}
	require.NoError(t, json.Unmarshal([]byte(channel.ConfigJSON), &cfg))
	require.Equal(t, "https://example.com/hook?machine={{.machine_id}}", cfg.URL)
	require.Equal(t, http.MethodPost, cfg.Method)
	require.Equal(t, map[string]string{"X-Test": "{{.machine_id}}"}, cfg.Headers)
	require.Equal(t, "{\"machine_id\":\"{{.machine_id}}\",\"metric\":\"{{.metric_type}}\"}", cfg.Body)
	require.Equal(t, uint(9), *cfg.ProxyID)
}

func TestAlertServiceUpsertTelegramChannelKeepsExistingTokenWhenMasked(t *testing.T) {
	channelStore := &stubNotificationChannelStore{
		channels: []model.NotificationChannel{
			{
				ChannelType: channelTypeTelegram,
				Enabled:     true,
				ConfigJSON:  `{"bot_token":"1234567890abcdef","chat_id":"10001"}`,
			},
		},
	}
	service := &AlertService{
		notificationChannelStore: channelStore,
		notificationProxyStore: &stubNotificationProxyStore{
			proxies: []model.NotificationProxy{
				{Base: model.Base{ID: 5}, Name: "proxy-5", ProxyType: "socks", URL: "socks5://127.0.0.1:1080"},
			},
		},
		log: zerolog.Nop(),
	}

	err := service.UpsertTelegramChannel(context.Background(), dto.UpsertTelegramChannelReq{
		Enabled:  true,
		BotToken: "1234...cdef",
		ChatID:   "20002",
		Message:  "{{machine_name}} {{actual_human_readable}}",
		ProxyID:  uintPtr(5),
	})
	require.NoError(t, err)
	require.Len(t, channelStore.channels, 1)

	var cfg struct {
		BotToken string `json:"bot_token"`
		ChatID   string `json:"chat_id"`
		Message  string `json:"message"`
		ProxyID  *uint  `json:"proxy_id"`
	}
	require.NoError(t, json.Unmarshal([]byte(channelStore.channels[0].ConfigJSON), &cfg))
	require.Equal(t, "1234567890abcdef", cfg.BotToken)
	require.Equal(t, "20002", cfg.ChatID)
	require.Equal(t, "{{machine_name}} {{actual_human_readable}}", cfg.Message)
	require.Equal(t, uint(5), *cfg.ProxyID)
}

func TestAlertServiceUpdateNotificationProxyKeepsMaskedURL(t *testing.T) {
	proxyStore := &stubNotificationProxyStore{
		proxies: []model.NotificationProxy{
			{
				Base:      model.Base{ID: 7},
				Name:      "old-proxy",
				ProxyType: "http",
				URL:       "http://user:secret@127.0.0.1:7890",
			},
		},
	}
	service := &AlertService{
		notificationProxyStore: proxyStore,
		log:                    zerolog.Nop(),
	}

	response, err := service.UpdateNotificationProxy(context.Background(), 7, dto.UpsertNotificationProxyReq{
		Name:      "new-proxy",
		ProxyType: "http",
		URL:       "http://user:xxxxx@127.0.0.1:7890",
	})
	require.NoError(t, err)
	require.Equal(t, "new-proxy", response.Name)
	require.Equal(t, "http://user:xxxxx@127.0.0.1:7890", response.URL)
	require.Equal(t, "http://user:secret@127.0.0.1:7890", proxyStore.proxies[0].URL)
}

func TestParseNotificationProxyURLNormalizesProxySchemes(t *testing.T) {
	httpProxyURL, err := parseNotificationProxyURL("http", "127.0.0.1:7890")
	require.NoError(t, err)
	require.Equal(t, "http", httpProxyURL.Scheme)
	require.Equal(t, "127.0.0.1", httpProxyURL.Hostname())

	socksProxyURL, err := parseNotificationProxyURL("socks", "socks://127.0.0.1:1080")
	require.NoError(t, err)
	require.Equal(t, "socks5", socksProxyURL.Scheme)
	require.Equal(t, "127.0.0.1", socksProxyURL.Hostname())

	_, err = parseNotificationProxyURL("socks", "http://127.0.0.1:7890")
	require.ErrorIs(t, err, ErrInvalidNotificationProxy)
}

func TestAlertServiceTestTelegramChannelExecutesRequest(t *testing.T) {
	channelStore := &stubNotificationChannelStore{
		channels: []model.NotificationChannel{
			{
				ChannelType: channelTypeTelegram,
				Enabled:     true,
				ConfigJSON:  `{"bot_token":"1234567890abcdef","chat_id":"10001","message":"old message"}`,
			},
		},
	}
	httpClient := &stubHTTPClient{
		statusCode: http.StatusOK,
		body:       `{"ok":true}`,
	}
	service := &AlertService{
		notificationChannelStore: channelStore,
		httpClient:               httpClient,
		machineStore: &stubAlertMachineStore{
			machines: map[uint]model.Machine{
				1: {
					Name: "prod-api-01",
					Host: "10.2.1.107",
				},
			},
		},
		log: zerolog.Nop(),
	}

	response, err := service.TestTelegramChannel(context.Background(), dto.UpsertTelegramChannelReq{
		BotToken: "1234...cdef",
		ChatID:   "20002",
		Message:  "machine={{machine_name}} host={{machine_host}} actual={{actual_human_readable}} key={{alert_key}}",
	})
	require.NoError(t, err)
	require.Len(t, httpClient.requests, 1)

	req := httpClient.requests[0]
	require.Equal(t, http.MethodPost, req.Method)
	require.Equal(t, "https://api.telegram.org/bot1234567890abcdef/sendMessage", req.URL.String())
	require.Equal(t, "application/json", req.Header.Get("Content-Type"))

	body, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"chat_id":"20002","text":"machine=prod-api-01 host=10.2.1.107 actual=1.500 GB key=test:telegram:alert"}`, string(body))
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, `{"ok":true}`, response.Body)
	require.Equal(t, "machine=prod-api-01 host=10.2.1.107 actual=1.500 GB key=test:telegram:alert", response.RenderedMessage)
}

func TestAlertServiceTestWebhookChannelExecutesRequest(t *testing.T) {
	httpClient := &stubHTTPClient{
		statusCode: http.StatusCreated,
		body:       "accepted",
	}
	service := &AlertService{
		httpClient: httpClient,
		machineStore: &stubAlertMachineStore{
			machines: map[uint]model.Machine{
				1: {
					Name: "prod-api-01",
					Host: "10.2.1.107",
				},
			},
		},
		log: zerolog.Nop(),
	}

	response, err := service.TestWebhookChannel(context.Background(), dto.UpsertWebhookChannelReq{
		Method: "POST",
		URL:    "https://example.com/hook?machine={{machine_name}}&host={{machine_host}}",
		Headers: map[string]string{
			"X-Machine-Name": "{{machine_name}}",
			"X-Machine-Host": "{{machine_host}}",
		},
		Body: "{\"machine_name\":\"{{machine_name}}\",\"machine_host\":\"{{machine_host}}\",\"metric\":\"{{metric_type}}\"}",
	})
	require.NoError(t, err)
	require.Len(t, httpClient.requests, 1)

	req := httpClient.requests[0]
	require.Equal(t, http.MethodPost, req.Method)
	require.Equal(t, "https://example.com/hook?machine=prod-api-01&host=10.2.1.107", req.URL.String())
	require.Equal(t, "prod-api-01", req.Header.Get("X-Machine-Name"))
	require.Equal(t, "10.2.1.107", req.Header.Get("X-Machine-Host"))
	require.Equal(t, "application/json", req.Header.Get("Content-Type"))

	body, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	require.JSONEq(t, "{\"machine_name\":\"prod-api-01\",\"machine_host\":\"10.2.1.107\",\"metric\":\"total\"}", string(body))

	var preview dto.TestWebhookChannelResp
	require.NoError(t, json.Unmarshal([]byte(response), &preview))
	require.Equal(t, http.StatusCreated, preview.StatusCode)
	require.Equal(t, "accepted", preview.Body)
	require.Equal(t, "https://example.com/hook?machine=prod-api-01&host=10.2.1.107", preview.RenderedURL)
	require.Equal(t, map[string]string{
		"X-Machine-Name": "prod-api-01",
		"X-Machine-Host": "10.2.1.107",
	}, preview.RenderedHeaders)
	require.JSONEq(t, "{\"machine_name\":\"prod-api-01\",\"machine_host\":\"10.2.1.107\",\"metric\":\"total\"}", preview.RenderedBody)
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
		httpClient: &stubHTTPClient{err: errors.New("network failed")},
		log:        zerolog.Nop(),
	}

	samples := []model.TrafficSample{
		{
			MachineID:  3,
			PeriodType: thresholdPeriodHourly,
			BucketTime: time.Now().UTC().Truncate(time.Hour).Add(-time.Hour),
			DownloadMB: 50,
		},
	}

	require.NoError(t, service.EvaluateSamples(context.Background(), 3, samples))
	require.Equal(t, alertNotifyStatusFailed, alertStore.list[0].NotifyStatus)
}
