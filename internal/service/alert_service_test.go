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
			BucketTime: time.Date(2026, 4, 25, 13, 0, 0, 0, time.UTC),
			TotalMB:    120,
		},
	}

	require.NoError(t, service.EvaluateSamples(context.Background(), 1, samples))
	require.NoError(t, service.EvaluateSamples(context.Background(), 1, samples))
	require.Len(t, alertStore.list, 1)
	require.Equal(t, alertNotifyStatusSkipped, alertStore.list[0].NotifyStatus)
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
				require.Equal(t, "2:daily:upload:1777075200", req.Header.Get("X-Alert-Key"))
				require.Equal(t, "application/json", req.Header.Get("Content-Type"))
				require.Equal(t, "https://example.com/hook?machine=2&metric=upload", req.URL.String())

				body, err := io.ReadAll(req.Body)
				require.NoError(t, err)
				require.JSONEq(t, `{"machine_id":2,"metric_type":"upload","actual_mb":5,"threshold_mb":1,"bucket_time":"2026-04-25T00:00:00Z"}`, string(body))

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       ioNopCloser{reader: strings.NewReader("ok")},
				}, nil
			},
		},
		log: zerolog.Nop(),
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
				require.Equal(t, "2:daily:upload:1777075200", req.Header.Get("X-Alert-Key"))
				require.Empty(t, req.Header.Get("Content-Type"))

				parsedURL, err := url.Parse(req.URL.String())
				require.NoError(t, err)
				require.Equal(t, "2", parsedURL.Query().Get("machine"))
				require.Equal(t, "upload", parsedURL.Query().Get("metric"))
				require.Equal(t, "2026-04-25T00:00:00Z", parsedURL.Query().Get("bucket"))

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
				require.Equal(t, "2:daily:upload:1777075200", req.Header.Get("X-Alert-Key"))
				require.Equal(t, "application/json", req.Header.Get("Content-Type"))
				require.Equal(t, "https://example.com/hook?machine=2&metric=upload", req.URL.String())

				body, err := io.ReadAll(req.Body)
				require.NoError(t, err)
				require.JSONEq(t, `{"machine_id":2,"metric_type":"upload","actual_mb":5,"threshold_mb":1,"bucket_time":"2026-04-25T00:00:00Z"}`, string(body))

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       ioNopCloser{reader: strings.NewReader("ok")},
				}, nil
			},
		},
		log: zerolog.Nop(),
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

func TestAlertServiceListNotificationChannelsIncludesWebhookTemplates(t *testing.T) {
	service := &AlertService{
		notificationChannelStore: &stubNotificationChannelStore{
			channels: []model.NotificationChannel{
				{
					ChannelType: channelTypeWebhook,
					Enabled:     true,
					ConfigJSON:  `{"url":"https://example.com/hook?machine={{.machine_id}}","method":"POST","headers":{"X-Test":"{{.machine_id}}"},"body":"{\"machine_id\":\"{{.machine_id}}\"}"}`,
				},
				{
					ChannelType: channelTypeTelegram,
					Enabled:     true,
					ConfigJSON:  `{"bot_token":"1234567890abcdef","chat_id":"10001"}`,
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

	telegram := channels[1]
	require.Equal(t, channelTypeTelegram, telegram.ChannelType)
	require.Equal(t, "10001", telegram.ChatID)
	require.NotEmpty(t, telegram.TokenMasked)
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
		log:                      zerolog.Nop(),
	}

	err := service.UpsertWebhookChannel(context.Background(), dto.UpsertWebhookChannelReq{
		Enabled: true,
		Method:  "post",
		URL:     "https://example.com/hook?machine={{.machine_id}}",
		Headers: map[string]string{
			"X-Test": "{{.machine_id}}",
		},
		Body: "{\"machine_id\":\"{{.machine_id}}\",\"metric\":\"{{.metric_type}}\"}",
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
	}
	require.NoError(t, json.Unmarshal([]byte(channel.ConfigJSON), &cfg))
	require.Equal(t, "https://example.com/hook?machine={{.machine_id}}", cfg.URL)
	require.Equal(t, http.MethodPost, cfg.Method)
	require.Equal(t, map[string]string{"X-Test": "{{.machine_id}}"}, cfg.Headers)
	require.Equal(t, "{\"machine_id\":\"{{.machine_id}}\",\"metric\":\"{{.metric_type}}\"}", cfg.Body)
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
			BucketTime: time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC),
			DownloadMB: 50,
		},
	}

	require.NoError(t, service.EvaluateSamples(context.Background(), 3, samples))
	require.Equal(t, alertNotifyStatusFailed, alertStore.list[0].NotifyStatus)
}
