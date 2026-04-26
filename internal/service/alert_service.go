package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"text/template"
	"time"

	"traffic-monitor/internal/dto"
	"traffic-monitor/internal/model"
	"traffic-monitor/internal/repo"

	"github.com/rs/zerolog"
	xproxy "golang.org/x/net/proxy"
)

const (
	alertNotifyStatusPending = "pending"
	alertNotifyStatusSent    = "sent"
	alertNotifyStatusFailed  = "failed"
	alertNotifyStatusSkipped = "skipped"
	channelTypeWebhook       = "webhook"
	channelTypeTelegram      = "telegram"
	defaultTelegramMessage   = "machine={{machine_name}} host={{machine_host}} period={{period_type}} metric={{metric_type}} actual={{actual_human_readable}} threshold={{threshold_human_readable}} bucket={{bucket_time_rfc3339}}"
)

var (
	ErrInvalidNotificationChannel = errors.New("invalid notification channel")
	ErrInvalidNotificationProxy   = errors.New("invalid notification proxy")
	ErrNotificationProxyNotFound  = errors.New("notification proxy not found")
)

type AlertStore interface {
	CreateIfAbsent(ctx context.Context, alert *model.Alert) (bool, error)
	UpdateNotifyStatus(ctx context.Context, alertID uint, notifyStatus string) error
	List(ctx context.Context, filter repo.AlertFilter) ([]model.Alert, int64, error)
}

type NotificationChannelStore interface {
	Upsert(ctx context.Context, channel *model.NotificationChannel) error
	List(ctx context.Context) ([]model.NotificationChannel, error)
}

type NotificationProxyStore interface {
	Create(ctx context.Context, notificationProxy *model.NotificationProxy) error
	Update(ctx context.Context, notificationProxy *model.NotificationProxy) error
	Delete(ctx context.Context, proxyID uint) error
	GetByID(ctx context.Context, proxyID uint) (*model.NotificationProxy, error)
	List(ctx context.Context) ([]model.NotificationProxy, error)
}

type AlertMachineStore interface {
	GetByID(ctx context.Context, machineID uint) (*model.Machine, error)
}

type NotificationDeliveryStore interface {
	Create(ctx context.Context, delivery *model.NotificationDelivery) error
	GetLatestByAlertIDs(ctx context.Context, alertIDs []uint) (map[uint]model.NotificationDelivery, error)
}

type ThresholdRuleProvider interface {
	ListEffectiveMachineRules(ctx context.Context, machineID uint) ([]dto.ThresholdRuleResp, error)
}

type NotificationHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type webhookChannelConfig struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
	ProxyID *uint             `json:"proxy_id,omitempty"`
}

type telegramChannelConfig struct {
	BotToken string `json:"bot_token"`
	ChatID   string `json:"chat_id"`
	Message  string `json:"message"`
	ProxyID  *uint  `json:"proxy_id,omitempty"`
}

type AlertService struct {
	alertStore                AlertStore
	notificationChannelStore  NotificationChannelStore
	notificationProxyStore    NotificationProxyStore
	notificationDeliveryStore NotificationDeliveryStore
	thresholdProvider         ThresholdRuleProvider
	machineStore              AlertMachineStore
	httpClient                NotificationHTTPClient
	log                       zerolog.Logger
}

func NewAlertService(alertStore *repo.AlertRepo, notificationChannelStore *repo.NotificationChannelRepo, notificationProxyStore *repo.NotificationProxyRepo, notificationDeliveryStore *repo.NotificationDeliveryRepo, thresholdProvider *ThresholdService, machineStore *repo.MachineRepo, log zerolog.Logger) *AlertService {
	return &AlertService{
		alertStore:                alertStore,
		notificationChannelStore:  notificationChannelStore,
		notificationProxyStore:    notificationProxyStore,
		notificationDeliveryStore: notificationDeliveryStore,
		thresholdProvider:         thresholdProvider,
		machineStore:              machineStore,
		httpClient:                &http.Client{Timeout: 10 * time.Second},
		log:                       log,
	}
}

func (service *AlertService) EvaluateSamples(ctx context.Context, machineID uint, samples []model.TrafficSample) error {
	rules, err := service.thresholdProvider.ListEffectiveMachineRules(ctx, machineID)
	if err != nil {
		return fmt.Errorf("list effective threshold rules: %w", err)
	}

	if len(rules) == 0 {
		return nil
	}

	ruleMap := make(map[string]dto.ThresholdRuleResp, len(rules))
	for _, rule := range rules {
		ruleMap[thresholdRuleKey(rule.PeriodType, rule.MetricType)] = rule
	}

	now := time.Now().UTC()
	for _, sample := range samples {
		if !isCompletedAlertPeriod(sample, now) {
			continue
		}

		if err := service.evaluateSample(ctx, sample, ruleMap); err != nil {
			return err
		}
	}

	return nil
}

func (service *AlertService) evaluateSample(ctx context.Context, sample model.TrafficSample, ruleMap map[string]dto.ThresholdRuleResp) error {
	metrics := []struct {
		metricType string
		actualMB   float64
	}{
		{metricType: thresholdMetricUpload, actualMB: sample.UploadMB},
		{metricType: thresholdMetricDownload, actualMB: sample.DownloadMB},
		{metricType: thresholdMetricTotal, actualMB: sample.TotalMB},
	}

	for _, metric := range metrics {
		rule, ok := ruleMap[thresholdRuleKey(sample.PeriodType, metric.metricType)]
		if !ok || !rule.Enabled || metric.actualMB <= rule.ThresholdMB {
			continue
		}

		alert := &model.Alert{
			MachineID:    sample.MachineID,
			PeriodType:   sample.PeriodType,
			MetricType:   metric.metricType,
			BucketTime:   sample.BucketTime,
			ThresholdMB:  rule.ThresholdMB,
			ActualMB:     metric.actualMB,
			AlertKey:     buildAlertKey(sample.MachineID, sample.PeriodType, metric.metricType, sample.BucketTime),
			NotifyStatus: alertNotifyStatusPending,
		}

		created, err := service.alertStore.CreateIfAbsent(ctx, alert)
		if err != nil {
			return fmt.Errorf("create alert: %w", err)
		}

		if !created {
			continue
		}

		if err := service.dispatchAlert(ctx, alert); err != nil {
			return err
		}
	}

	return nil
}

func (service *AlertService) dispatchAlert(ctx context.Context, alert *model.Alert) error {
	channels, err := service.notificationChannelStore.List(ctx)
	if err != nil {
		return fmt.Errorf("list notification channels: %w", err)
	}

	enabledChannels := make([]model.NotificationChannel, 0, len(channels))
	for _, channel := range channels {
		if channel.Enabled {
			enabledChannels = append(enabledChannels, channel)
		}
	}

	if len(enabledChannels) == 0 {
		service.log.Info().
			Uint("machine_id", alert.MachineID).
			Str("period_type", alert.PeriodType).
			Str("metric_type", alert.MetricType).
			Msg("alert skipped without enabled notification channels")
		return service.alertStore.UpdateNotifyStatus(ctx, alert.ID, alertNotifyStatusSkipped)
	}

	allSucceeded := true
	for _, channel := range enabledChannels {
		delivery := &model.NotificationDelivery{
			AlertID:     alert.ID,
			ChannelType: channel.ChannelType,
		}

		responseExcerpt, sendErr := service.sendNotification(ctx, channel, alert)
		if sendErr != nil {
			allSucceeded = false
			delivery.Success = false
			delivery.ErrorMessage = trimMessage(sendErr.Error())
		} else {
			delivery.Success = true
			delivery.ResponseExcerpt = trimMessage(responseExcerpt)
		}

		if err := service.notificationDeliveryStore.Create(ctx, delivery); err != nil {
			return fmt.Errorf("create notification delivery: %w", err)
		}
	}

	if allSucceeded {
		return service.alertStore.UpdateNotifyStatus(ctx, alert.ID, alertNotifyStatusSent)
	}

	return service.alertStore.UpdateNotifyStatus(ctx, alert.ID, alertNotifyStatusFailed)
}

func (service *AlertService) sendNotification(ctx context.Context, channel model.NotificationChannel, alert *model.Alert) (string, error) {
	switch channel.ChannelType {
	case channelTypeWebhook:
		return service.sendWebhook(ctx, channel.ConfigJSON, alert)
	case channelTypeTelegram:
		return service.sendTelegram(ctx, channel.ConfigJSON, alert)
	default:
		return "", ErrInvalidNotificationChannel
	}
}

func (service *AlertService) sendWebhook(ctx context.Context, configJSON string, alert *model.Alert) (string, error) {
	cfg, err := decodeWebhookChannelConfig(configJSON)
	if err != nil {
		return "", err
	}

	req, _, err := service.renderWebhookRequestFromConfig(ctx, cfg, alert)
	if err != nil {
		return "", err
	}

	resp, err := service.doNotificationRequest(ctx, req, cfg.ProxyID)
	if err != nil {
		return "", fmt.Errorf("send webhook request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if resp.StatusCode >= 300 {
		return string(body), fmt.Errorf("webhook status %d", resp.StatusCode)
	}

	return string(body), nil
}

func (service *AlertService) sendTelegram(ctx context.Context, configJSON string, alert *model.Alert) (string, error) {
	var cfg telegramChannelConfig

	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return "", fmt.Errorf("decode telegram config: %w", err)
	}

	req, _, err := service.renderTelegramRequest(ctx, cfg, alert)
	if err != nil {
		return "", err
	}

	resp, err := service.doNotificationRequest(ctx, req, cfg.ProxyID)
	if err != nil {
		return "", fmt.Errorf("send telegram request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if resp.StatusCode >= 300 {
		return string(body), fmt.Errorf("telegram status %d", resp.StatusCode)
	}

	return string(body), nil
}

func (service *AlertService) doNotificationRequest(ctx context.Context, req *http.Request, proxyID *uint) (*http.Response, error) {
	normalizedProxyID := normalizeNotificationProxyID(proxyID)
	if normalizedProxyID == nil {
		return service.httpClient.Do(req)
	}

	if service.notificationProxyStore == nil {
		return nil, ErrInvalidNotificationProxy
	}

	notificationProxy, err := service.notificationProxyStore.GetByID(ctx, *normalizedProxyID)
	if err != nil {
		if repo.IsRecordNotFound(err) {
			return nil, ErrInvalidNotificationProxy
		}

		return nil, fmt.Errorf("get notification proxy: %w", err)
	}
	if notificationProxy == nil {
		return nil, ErrInvalidNotificationProxy
	}

	httpClient, err := buildNotificationProxyHTTPClient(notificationProxy)
	if err != nil {
		return nil, err
	}

	return httpClient.Do(req)
}

func (service *AlertService) validateNotificationProxyID(ctx context.Context, proxyID *uint) error {
	normalizedProxyID := normalizeNotificationProxyID(proxyID)
	if normalizedProxyID == nil {
		return nil
	}
	if service.notificationProxyStore == nil {
		return ErrInvalidNotificationProxy
	}

	notificationProxy, err := service.notificationProxyStore.GetByID(ctx, *normalizedProxyID)
	if err != nil {
		if repo.IsRecordNotFound(err) {
			return ErrInvalidNotificationProxy
		}

		return fmt.Errorf("get notification proxy: %w", err)
	}
	if notificationProxy == nil {
		return ErrInvalidNotificationProxy
	}

	return nil
}

func (service *AlertService) ListAlerts(ctx context.Context, query dto.ListAlertsQuery) (dto.AlertListResp, error) {
	alerts, total, err := service.alertStore.List(ctx, repo.AlertFilter{
		MachineID:  query.MachineID,
		PeriodType: query.PeriodType,
		Page:       query.Page,
		PageSize:   query.PageSize,
	})
	if err != nil {
		return dto.AlertListResp{}, fmt.Errorf("list alerts: %w", err)
	}

	alertIDs := make([]uint, 0, len(alerts))
	for _, alert := range alerts {
		alertIDs = append(alertIDs, alert.ID)
	}

	latestDeliveries, err := service.notificationDeliveryStore.GetLatestByAlertIDs(ctx, alertIDs)
	if err != nil {
		return dto.AlertListResp{}, fmt.Errorf("list latest notification deliveries: %w", err)
	}

	items := make([]dto.AlertResp, 0, len(alerts))
	for _, alert := range alerts {
		var notifiedAt *time.Time
		if delivery, ok := latestDeliveries[alert.ID]; ok {
			deliveryCreatedAt := delivery.CreatedAt
			notifiedAt = &deliveryCreatedAt
		}

		items = append(items, dto.AlertResp{
			ID:           alert.ID,
			MachineID:    alert.MachineID,
			PeriodType:   alert.PeriodType,
			MetricType:   alert.MetricType,
			BucketTime:   alert.BucketTime,
			ThresholdMB:  alert.ThresholdMB,
			ActualMB:     alert.ActualMB,
			NotifyStatus: alert.NotifyStatus,
			NotifiedAt:   notifiedAt,
			CreatedAt:    alert.CreatedAt,
		})
	}

	return dto.AlertListResp{
		Items: items,
		Total: total,
	}, nil
}

func (service *AlertService) ListNotificationChannels(ctx context.Context) ([]dto.NotificationChannelResp, error) {
	channels, err := service.notificationChannelStore.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list notification channels: %w", err)
	}

	result := make([]dto.NotificationChannelResp, 0, len(channels))
	for _, channel := range channels {
		switch channel.ChannelType {
		case channelTypeWebhook:
			var cfg webhookChannelConfig
			_ = json.Unmarshal([]byte(channel.ConfigJSON), &cfg)
			method := strings.ToUpper(strings.TrimSpace(cfg.Method))
			if method == "" {
				method = http.MethodPost
			}
			result = append(result, dto.NotificationChannelResp{
				ChannelType: channel.ChannelType,
				Enabled:     channel.Enabled,
				Configured:  strings.TrimSpace(cfg.URL) != "",
				Method:      method,
				URL:         cfg.URL,
				Headers:     cfg.Headers,
				Body:        cfg.Body,
				ProxyID:     cfg.ProxyID,
			})
		case channelTypeTelegram:
			var cfg telegramChannelConfig
			_ = json.Unmarshal([]byte(channel.ConfigJSON), &cfg)
			message := strings.TrimSpace(cfg.Message)
			if message == "" {
				message = defaultTelegramMessage
			}
			result = append(result, dto.NotificationChannelResp{
				ChannelType: channel.ChannelType,
				Enabled:     channel.Enabled,
				Configured:  strings.TrimSpace(cfg.BotToken) != "" && strings.TrimSpace(cfg.ChatID) != "",
				ChatID:      cfg.ChatID,
				Message:     message,
				TokenMasked: maskToken(cfg.BotToken),
				ProxyID:     cfg.ProxyID,
			})
		}
	}

	return result, nil
}

func (service *AlertService) ListNotificationProxies(ctx context.Context) ([]dto.NotificationProxyResp, error) {
	notificationProxies, err := service.notificationProxyStore.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list notification proxies: %w", err)
	}

	result := make([]dto.NotificationProxyResp, 0, len(notificationProxies))
	for _, notificationProxy := range notificationProxies {
		result = append(result, toNotificationProxyResp(&notificationProxy))
	}

	return result, nil
}

func (service *AlertService) CreateNotificationProxy(ctx context.Context, req dto.UpsertNotificationProxyReq) (dto.NotificationProxyResp, error) {
	notificationProxy, err := service.buildNotificationProxy(ctx, 0, req)
	if err != nil {
		return dto.NotificationProxyResp{}, err
	}

	if err := service.notificationProxyStore.Create(ctx, notificationProxy); err != nil {
		return dto.NotificationProxyResp{}, fmt.Errorf("create notification proxy: %w", err)
	}

	return toNotificationProxyResp(notificationProxy), nil
}

func (service *AlertService) UpdateNotificationProxy(ctx context.Context, proxyID uint, req dto.UpsertNotificationProxyReq) (dto.NotificationProxyResp, error) {
	notificationProxy, err := service.buildNotificationProxy(ctx, proxyID, req)
	if err != nil {
		return dto.NotificationProxyResp{}, err
	}

	if err := service.notificationProxyStore.Update(ctx, notificationProxy); err != nil {
		if repo.IsRecordNotFound(err) {
			return dto.NotificationProxyResp{}, ErrNotificationProxyNotFound
		}

		return dto.NotificationProxyResp{}, fmt.Errorf("update notification proxy: %w", err)
	}

	return toNotificationProxyResp(notificationProxy), nil
}

func (service *AlertService) DeleteNotificationProxy(ctx context.Context, proxyID uint) error {
	if err := service.notificationProxyStore.Delete(ctx, proxyID); err != nil {
		if repo.IsRecordNotFound(err) {
			return ErrNotificationProxyNotFound
		}

		return fmt.Errorf("delete notification proxy: %w", err)
	}

	return nil
}

func (service *AlertService) UpsertWebhookChannel(ctx context.Context, req dto.UpsertWebhookChannelReq) error {
	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodPost
	}
	if err := service.validateNotificationProxyID(ctx, req.ProxyID); err != nil {
		return err
	}

	configJSON, err := json.Marshal(webhookChannelConfig{
		URL:     strings.TrimSpace(req.URL),
		Method:  method,
		Headers: req.Headers,
		Body:    req.Body,
		ProxyID: normalizeNotificationProxyID(req.ProxyID),
	})
	if err != nil {
		return fmt.Errorf("marshal webhook channel config: %w", err)
	}

	channel := &model.NotificationChannel{
		ChannelType: channelTypeWebhook,
		Enabled:     req.Enabled,
		ConfigJSON:  string(configJSON),
	}

	if err := service.notificationChannelStore.Upsert(ctx, channel); err != nil {
		return fmt.Errorf("upsert webhook channel: %w", err)
	}

	return nil
}

func (service *AlertService) TestWebhookChannel(ctx context.Context, req dto.UpsertWebhookChannelReq) (string, error) {
	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodPost
	}
	if method != http.MethodPost && method != http.MethodGet {
		return "", ErrInvalidNotificationChannel
	}

	alert := &model.Alert{
		MachineID:    1,
		PeriodType:   thresholdPeriodHourly,
		MetricType:   thresholdMetricTotal,
		BucketTime:   time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC),
		ThresholdMB:  1024,
		ActualMB:     1536,
		AlertKey:     "test:webhook:alert",
		NotifyStatus: alertNotifyStatusPending,
	}

	configJSON, err := json.Marshal(webhookChannelConfig{
		URL:     strings.TrimSpace(req.URL),
		Method:  method,
		Headers: req.Headers,
		Body:    req.Body,
		ProxyID: normalizeNotificationProxyID(req.ProxyID),
	})
	if err != nil {
		return "", fmt.Errorf("marshal webhook test config: %w", err)
	}

	cfg, err := decodeWebhookChannelConfig(string(configJSON))
	if err != nil {
		return "", err
	}

	httpRequest, previewJSON, err := service.renderWebhookRequestFromConfig(ctx, cfg, alert)
	if err != nil {
		return "", err
	}

	resp, err := service.doNotificationRequest(ctx, httpRequest, cfg.ProxyID)
	if err != nil {
		return "", fmt.Errorf("send webhook request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil {
		return "", fmt.Errorf("read webhook response: %w", err)
	}

	var preview struct {
		Method  string            `json:"method"`
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
		Body    string            `json:"body"`
	}
	if err := json.Unmarshal([]byte(previewJSON), &preview); err != nil {
		return "", fmt.Errorf("decode webhook preview: %w", err)
	}

	resultJSON, err := json.Marshal(dto.TestWebhookChannelResp{
		StatusCode:      resp.StatusCode,
		Body:            string(responseBody),
		RenderedURL:     preview.URL,
		RenderedHeaders: preview.Headers,
		RenderedBody:    preview.Body,
	})
	if err != nil {
		return "", fmt.Errorf("marshal webhook test response: %w", err)
	}

	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("webhook status %d: %s", resp.StatusCode, string(responseBody))
	}

	return string(resultJSON), nil
}

func (service *AlertService) UpsertTelegramChannel(ctx context.Context, req dto.UpsertTelegramChannelReq) error {
	cfg, err := service.buildTelegramChannelConfig(ctx, req)
	if err != nil {
		return err
	}

	configJSON, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal telegram channel config: %w", err)
	}

	channel := &model.NotificationChannel{
		ChannelType: channelTypeTelegram,
		Enabled:     req.Enabled,
		ConfigJSON:  string(configJSON),
	}

	if err := service.notificationChannelStore.Upsert(ctx, channel); err != nil {
		return fmt.Errorf("upsert telegram channel: %w", err)
	}

	return nil
}

func (service *AlertService) TestTelegramChannel(ctx context.Context, req dto.UpsertTelegramChannelReq) (dto.TestTelegramChannelResp, error) {
	cfg, err := service.buildTelegramChannelConfig(ctx, req)
	if err != nil {
		return dto.TestTelegramChannelResp{}, err
	}

	alert := &model.Alert{
		MachineID:    1,
		PeriodType:   thresholdPeriodHourly,
		MetricType:   thresholdMetricTotal,
		BucketTime:   time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC),
		ThresholdMB:  1024,
		ActualMB:     1536,
		AlertKey:     "test:telegram:alert",
		NotifyStatus: alertNotifyStatusPending,
	}

	httpRequest, renderedMessage, err := service.renderTelegramRequest(ctx, cfg, alert)
	if err != nil {
		return dto.TestTelegramChannelResp{}, err
	}

	resp, err := service.doNotificationRequest(ctx, httpRequest, cfg.ProxyID)
	if err != nil {
		return dto.TestTelegramChannelResp{}, fmt.Errorf("send telegram request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil {
		return dto.TestTelegramChannelResp{}, fmt.Errorf("read telegram response: %w", err)
	}

	result := dto.TestTelegramChannelResp{
		StatusCode:      resp.StatusCode,
		Body:            string(responseBody),
		RenderedMessage: renderedMessage,
	}
	if resp.StatusCode >= 300 {
		return dto.TestTelegramChannelResp{}, fmt.Errorf("telegram status %d: %s", resp.StatusCode, string(responseBody))
	}

	return result, nil
}

func (service *AlertService) buildTelegramChannelConfig(ctx context.Context, req dto.UpsertTelegramChannelReq) (telegramChannelConfig, error) {
	botToken := strings.TrimSpace(req.BotToken)
	chatID := strings.TrimSpace(req.ChatID)
	message := strings.TrimSpace(req.Message)
	if message == "" {
		message = defaultTelegramMessage
	}
	if err := service.validateNotificationProxyID(ctx, req.ProxyID); err != nil {
		return telegramChannelConfig{}, err
	}

	if service.notificationChannelStore != nil {
		channels, err := service.notificationChannelStore.List(ctx)
		if err != nil {
			return telegramChannelConfig{}, fmt.Errorf("list notification channels: %w", err)
		}

		for _, channel := range channels {
			if channel.ChannelType != channelTypeTelegram {
				continue
			}

			var cfg telegramChannelConfig
			_ = json.Unmarshal([]byte(channel.ConfigJSON), &cfg)
			existingBotToken := strings.TrimSpace(cfg.BotToken)
			if existingBotToken != "" && (botToken == "" || botToken == maskToken(existingBotToken)) {
				botToken = existingBotToken
			}
			break
		}
	}

	return telegramChannelConfig{
		BotToken: botToken,
		ChatID:   chatID,
		Message:  message,
		ProxyID:  normalizeNotificationProxyID(req.ProxyID),
	}, nil
}

func (service *AlertService) buildNotificationProxy(ctx context.Context, proxyID uint, req dto.UpsertNotificationProxyReq) (*model.NotificationProxy, error) {
	name := strings.TrimSpace(req.Name)
	proxyType := normalizeNotificationProxyType(req.ProxyType)
	proxyURL := strings.TrimSpace(req.URL)

	var existing *model.NotificationProxy
	if proxyID != 0 {
		notificationProxy, err := service.notificationProxyStore.GetByID(ctx, proxyID)
		if err != nil {
			if repo.IsRecordNotFound(err) {
				return nil, ErrNotificationProxyNotFound
			}

			return nil, fmt.Errorf("get notification proxy: %w", err)
		}
		existing = notificationProxy
		if existing == nil {
			return nil, ErrNotificationProxyNotFound
		}
		if proxyURL == maskProxyURL(existing.URL) {
			proxyURL = existing.URL
		}
	}

	if name == "" || proxyType == "" || proxyURL == "" {
		return nil, ErrInvalidNotificationProxy
	}
	parsedProxyURL, err := parseNotificationProxyURL(proxyType, proxyURL)
	if err != nil {
		return nil, err
	}
	proxyURL = parsedProxyURL.String()

	notificationProxy := &model.NotificationProxy{
		Name:      name,
		ProxyType: proxyType,
		URL:       proxyURL,
	}
	if existing != nil {
		notificationProxy.Base = existing.Base
	}
	if proxyID != 0 {
		notificationProxy.ID = proxyID
	}

	return notificationProxy, nil
}

func (service *AlertService) renderTelegramRequest(ctx context.Context, cfg telegramChannelConfig, alert *model.Alert) (*http.Request, string, error) {
	botToken := strings.TrimSpace(cfg.BotToken)
	chatID := strings.TrimSpace(cfg.ChatID)
	if botToken == "" || chatID == "" {
		return nil, "", ErrInvalidNotificationChannel
	}

	messageTemplate := strings.TrimSpace(cfg.Message)
	if messageTemplate == "" {
		messageTemplate = defaultTelegramMessage
	}

	renderedMessage, err := renderWebhookTemplate(messageTemplate, service.buildWebhookTemplateData(ctx, alert))
	if err != nil {
		return nil, "", fmt.Errorf("render telegram message template: %w", err)
	}

	bodyPayload, err := json.Marshal(map[string]string{
		"chat_id": chatID,
		"text":    renderedMessage,
	})
	if err != nil {
		return nil, "", fmt.Errorf("marshal telegram payload: %w", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyPayload))
	if err != nil {
		return nil, "", fmt.Errorf("build telegram request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	return req, renderedMessage, nil
}

func buildAlertKey(machineID uint, periodType string, metricType string, bucketTime time.Time) string {
	return fmt.Sprintf("%d:%s:%s:%d", machineID, periodType, metricType, bucketTime.UTC().Unix())
}

func isCompletedAlertPeriod(sample model.TrafficSample, now time.Time) bool {
	bucketTime := sample.BucketTime.UTC()
	currentTime := now.UTC()

	switch sample.PeriodType {
	case thresholdPeriodHourly:
		currentHourStart := currentTime.Truncate(time.Hour)
		return bucketTime.Before(currentHourStart)
	case thresholdPeriodDaily:
		currentDayStart := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, time.UTC)
		return bucketTime.Before(currentDayStart)
	default:
		return true
	}
}

func (service *AlertService) buildWebhookTemplateData(ctx context.Context, alert *model.Alert) map[string]interface{} {
	machineName := "test-machine"
	machineHost := "127.0.0.1"
	if service.machineStore != nil && alert.MachineID != 0 {
		machine, err := service.machineStore.GetByID(ctx, alert.MachineID)
		if err == nil && machine != nil {
			if strings.TrimSpace(machine.Name) != "" {
				machineName = machine.Name
			}
			if strings.TrimSpace(machine.Host) != "" {
				machineHost = machine.Host
			}
		}
	}

	return map[string]interface{}{
		"machine_id":                 alert.MachineID,
		"machine_name":               machineName,
		"machine_host":               machineHost,
		"period_type":                alert.PeriodType,
		"metric_type":                alert.MetricType,
		"bucket_time":                alert.BucketTime,
		"bucket_time_rfc3339":        alert.BucketTime.UTC().Format(time.RFC3339),
		"threshold_mb":               alert.ThresholdMB,
		"threshold_human_readable":   formatTrafficMB(alert.ThresholdMB),
		"actual_mb":                  alert.ActualMB,
		"actual_human_readable":      formatTrafficMB(alert.ActualMB),
		"alert_key":                  alert.AlertKey,
	}
}

func (service *AlertService) renderWebhookRequest(ctx context.Context, configJSON string, alert *model.Alert) (*http.Request, string, error) {
	cfg, err := decodeWebhookChannelConfig(configJSON)
	if err != nil {
		return nil, "", err
	}

	return service.renderWebhookRequestFromConfig(ctx, cfg, alert)
}

func (service *AlertService) renderWebhookRequestFromConfig(ctx context.Context, cfg webhookChannelConfig, alert *model.Alert) (*http.Request, string, error) {
	if strings.TrimSpace(cfg.URL) == "" {
		return nil, "", ErrInvalidNotificationChannel
	}

	method := strings.ToUpper(strings.TrimSpace(cfg.Method))
	if method == "" {
		method = http.MethodPost
	}
	if method != http.MethodPost && method != http.MethodGet {
		return nil, "", ErrInvalidNotificationChannel
	}

	templateData := service.buildWebhookTemplateData(ctx, alert)

	renderedURL, err := renderWebhookTemplate(cfg.URL, templateData)
	if err != nil {
		return nil, "", fmt.Errorf("render webhook url template: %w", err)
	}

	renderedHeaders, err := renderWebhookHeaders(cfg.Headers, templateData)
	if err != nil {
		return nil, "", fmt.Errorf("render webhook headers template: %w", err)
	}

	defaultPayload, err := json.Marshal(map[string]interface{}{
		"machine_id":               alert.MachineID,
		"machine_name":             templateData["machine_name"],
		"machine_host":             templateData["machine_host"],
		"period_type":              alert.PeriodType,
		"metric_type":              alert.MetricType,
		"bucket_time":              alert.BucketTime,
		"threshold_mb":             alert.ThresholdMB,
		"threshold_human_readable": templateData["threshold_human_readable"],
		"actual_mb":                alert.ActualMB,
		"actual_human_readable":    templateData["actual_human_readable"],
		"alert_key":                alert.AlertKey,
	})
	if err != nil {
		return nil, "", fmt.Errorf("marshal webhook payload: %w", err)
	}

	renderedBody := ""
	var req *http.Request
	if method == http.MethodGet {
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, renderedURL, nil)
	} else {
		requestBody := defaultPayload
		if strings.TrimSpace(cfg.Body) != "" {
			renderedBody, err = renderWebhookTemplate(cfg.Body, templateData)
			if err != nil {
				return nil, "", fmt.Errorf("render webhook body template: %w", err)
			}
			requestBody = []byte(renderedBody)
		} else {
			renderedBody = string(defaultPayload)
		}

		req, err = http.NewRequestWithContext(ctx, http.MethodPost, renderedURL, bytes.NewReader(requestBody))
	}
	if err != nil {
		return nil, "", fmt.Errorf("build webhook request: %w", err)
	}

	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range renderedHeaders {
		req.Header.Set(key, value)
	}

	previewPayload, err := json.Marshal(map[string]interface{}{
		"method":  method,
		"url":     renderedURL,
		"headers": renderedHeaders,
		"body":    renderedBody,
	})
	if err != nil {
		return nil, "", fmt.Errorf("marshal webhook preview: %w", err)
	}

	return req, string(previewPayload), nil
}

func decodeWebhookChannelConfig(configJSON string) (webhookChannelConfig, error) {
	var cfg webhookChannelConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return webhookChannelConfig{}, fmt.Errorf("decode webhook config: %w", err)
	}

	cfg.ProxyID = normalizeNotificationProxyID(cfg.ProxyID)
	return cfg, nil
}

func buildNotificationProxyHTTPClient(notificationProxy *model.NotificationProxy) (*http.Client, error) {
	proxyURL, err := parseNotificationProxyURL(notificationProxy.ProxyType, notificationProxy.URL)
	if err != nil {
		return nil, err
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	switch normalizeNotificationProxyType(notificationProxy.ProxyType) {
	case "http":
		transport.Proxy = http.ProxyURL(proxyURL)
	case "socks":
		dialer, err := xproxy.FromURL(proxyURL, xproxy.Direct)
		if err != nil {
			return nil, ErrInvalidNotificationProxy
		}
		contextDialer, ok := dialer.(xproxy.ContextDialer)
		if !ok {
			return nil, ErrInvalidNotificationProxy
		}
		transport.Proxy = nil
		transport.DialContext = func(ctx context.Context, network string, address string) (net.Conn, error) {
			return contextDialer.DialContext(ctx, network, address)
		}
	default:
		return nil, ErrInvalidNotificationProxy
	}

	return &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
	}, nil
}

func parseNotificationProxyURL(proxyType string, rawURL string) (*url.URL, error) {
	normalizedProxyType := normalizeNotificationProxyType(proxyType)
	normalizedURL := strings.TrimSpace(rawURL)
	if normalizedProxyType == "" || normalizedURL == "" {
		return nil, ErrInvalidNotificationProxy
	}

	if !strings.Contains(normalizedURL, "://") {
		switch normalizedProxyType {
		case "http":
			normalizedURL = "http://" + normalizedURL
		case "socks":
			normalizedURL = "socks5://" + normalizedURL
		default:
			return nil, ErrInvalidNotificationProxy
		}
	}

	parsedURL, err := url.Parse(normalizedURL)
	if err != nil {
		return nil, ErrInvalidNotificationProxy
	}
	if parsedURL.Hostname() == "" {
		return nil, ErrInvalidNotificationProxy
	}

	parsedURL.Scheme = strings.ToLower(parsedURL.Scheme)
	switch normalizedProxyType {
	case "http":
		if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			return nil, ErrInvalidNotificationProxy
		}
	case "socks":
		if parsedURL.Scheme == "socks" {
			parsedURL.Scheme = "socks5"
		}
		if parsedURL.Scheme != "socks5" && parsedURL.Scheme != "socks5h" {
			return nil, ErrInvalidNotificationProxy
		}
	default:
		return nil, ErrInvalidNotificationProxy
	}

	return parsedURL, nil
}

func normalizeNotificationProxyType(proxyType string) string {
	switch strings.ToLower(strings.TrimSpace(proxyType)) {
	case "http", "https":
		return "http"
	case "socks", "socks5", "socks5h":
		return "socks"
	default:
		return ""
	}
}

func normalizeNotificationProxyID(proxyID *uint) *uint {
	if proxyID == nil || *proxyID == 0 {
		return nil
	}

	normalizedProxyID := *proxyID
	return &normalizedProxyID
}

func toNotificationProxyResp(notificationProxy *model.NotificationProxy) dto.NotificationProxyResp {
	return dto.NotificationProxyResp{
		ID:        notificationProxy.ID,
		Name:      notificationProxy.Name,
		ProxyType: notificationProxy.ProxyType,
		URL:       maskProxyURL(notificationProxy.URL),
		CreatedAt: notificationProxy.CreatedAt,
		UpdatedAt: notificationProxy.UpdatedAt,
	}
}

func renderWebhookTemplate(raw string, data map[string]interface{}) (string, error) {
	normalizedTemplate := normalizeWebhookTemplate(raw)

	tmpl, err := template.New("webhook").Option("missingkey=error").Parse(normalizedTemplate)
	if err != nil {
		return "", err
	}

	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, data); err != nil {
		return "", err
	}

	return rendered.String(), nil
}

func normalizeWebhookTemplate(raw string) string {
	replacer := strings.NewReplacer(
		"{{machine_id}}", "{{.machine_id}}",
		"{{machine_name}}", "{{.machine_name}}",
		"{{machine_host}}", "{{.machine_host}}",
		"{{period_type}}", "{{.period_type}}",
		"{{metric_type}}", "{{.metric_type}}",
		"{{bucket_time}}", "{{.bucket_time}}",
		"{{bucket_time_rfc3339}}", "{{.bucket_time_rfc3339}}",
		"{{threshold_mb}}", "{{.threshold_mb}}",
		"{{threshold_human_readable}}", "{{.threshold_human_readable}}",
		"{{actual_mb}}", "{{.actual_mb}}",
		"{{actual_human_readable}}", "{{.actual_human_readable}}",
		"{{alert_key}}", "{{.alert_key}}",
	)

	return replacer.Replace(raw)
}

func renderWebhookHeaders(headers map[string]string, data map[string]interface{}) (map[string]string, error) {
	if len(headers) == 0 {
		return map[string]string{}, nil
	}

	rendered := make(map[string]string, len(headers))
	for key, value := range headers {
		renderedValue, err := renderWebhookTemplate(value, data)
		if err != nil {
			return nil, fmt.Errorf("render header %q: %w", key, err)
		}
		rendered[key] = renderedValue
	}

	return rendered, nil
}

func trimMessage(message string) string {
	if len(message) <= 512 {
		return message
	}

	return message[:512]
}

func maskToken(token string) string {
	if len(token) <= 8 {
		return token
	}

	return token[:4] + "..." + token[len(token)-4:]
}

func maskProxyURL(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil || parsedURL.User == nil {
		return rawURL
	}

	username := parsedURL.User.Username()
	if _, ok := parsedURL.User.Password(); ok {
		parsedURL.User = url.UserPassword(username, "xxxxx")
	}

	return parsedURL.String()
}

func formatTrafficMB(valueMB float64) string {
	if valueMB >= 1024*1024 {
		return fmt.Sprintf("%.3f TB", valueMB/(1024*1024))
	}

	if valueMB >= 1024 {
		return fmt.Sprintf("%.3f GB", valueMB/1024)
	}

	return fmt.Sprintf("%.3f MB", valueMB)
}
