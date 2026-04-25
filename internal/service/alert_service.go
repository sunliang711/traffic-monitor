package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"traffic-monitor/internal/dto"
	"traffic-monitor/internal/model"
	"traffic-monitor/internal/repo"

	"github.com/rs/zerolog"
)

const (
	alertNotifyStatusPending = "pending"
	alertNotifyStatusSent    = "sent"
	alertNotifyStatusFailed  = "failed"
	alertNotifyStatusSkipped = "skipped"
	channelTypeWebhook       = "webhook"
	channelTypeTelegram      = "telegram"
)

var (
	ErrInvalidNotificationChannel = errors.New("invalid notification channel")
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

type NotificationDeliveryStore interface {
	Create(ctx context.Context, delivery *model.NotificationDelivery) error
}

type ThresholdRuleProvider interface {
	ListEffectiveMachineRules(ctx context.Context, machineID uint) ([]dto.ThresholdRuleResp, error)
}

type NotificationHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type AlertService struct {
	alertStore                AlertStore
	notificationChannelStore  NotificationChannelStore
	notificationDeliveryStore NotificationDeliveryStore
	thresholdProvider         ThresholdRuleProvider
	httpClient                NotificationHTTPClient
	log                       zerolog.Logger
}

func NewAlertService(alertStore *repo.AlertRepo, notificationChannelStore *repo.NotificationChannelRepo, notificationDeliveryStore *repo.NotificationDeliveryRepo, thresholdProvider *ThresholdService, log zerolog.Logger) *AlertService {
	return &AlertService{
		alertStore:                alertStore,
		notificationChannelStore:  notificationChannelStore,
		notificationDeliveryStore: notificationDeliveryStore,
		thresholdProvider:         thresholdProvider,
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

	for _, sample := range samples {
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
	var cfg struct {
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
	}

	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return "", fmt.Errorf("decode webhook config: %w", err)
	}

	if strings.TrimSpace(cfg.URL) == "" {
		return "", ErrInvalidNotificationChannel
	}

	payload, err := json.Marshal(map[string]interface{}{
		"machine_id":   alert.MachineID,
		"period_type":  alert.PeriodType,
		"metric_type":  alert.MetricType,
		"bucket_time":  alert.BucketTime,
		"threshold_mb": alert.ThresholdMB,
		"actual_mb":    alert.ActualMB,
		"alert_key":    alert.AlertKey,
	})
	if err != nil {
		return "", fmt.Errorf("marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.URL, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("build webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for key, value := range cfg.Headers {
		req.Header.Set(key, value)
	}

	resp, err := service.httpClient.Do(req)
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
	var cfg struct {
		BotToken string `json:"bot_token"`
		ChatID   string `json:"chat_id"`
	}

	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return "", fmt.Errorf("decode telegram config: %w", err)
	}

	if strings.TrimSpace(cfg.BotToken) == "" || strings.TrimSpace(cfg.ChatID) == "" {
		return "", ErrInvalidNotificationChannel
	}

	bodyPayload, err := json.Marshal(map[string]string{
		"chat_id": cfg.ChatID,
		"text": fmt.Sprintf("machine=%d period=%s metric=%s actual=%.3fMB threshold=%.3fMB bucket=%s",
			alert.MachineID, alert.PeriodType, alert.MetricType, alert.ActualMB, alert.ThresholdMB, alert.BucketTime.Format(time.RFC3339)),
	})
	if err != nil {
		return "", fmt.Errorf("marshal telegram payload: %w", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cfg.BotToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyPayload))
	if err != nil {
		return "", fmt.Errorf("build telegram request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := service.httpClient.Do(req)
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

	items := make([]dto.AlertResp, 0, len(alerts))
	for _, alert := range alerts {
		items = append(items, dto.AlertResp{
			ID:           alert.ID,
			MachineID:    alert.MachineID,
			PeriodType:   alert.PeriodType,
			MetricType:   alert.MetricType,
			BucketTime:   alert.BucketTime,
			ThresholdMB:  alert.ThresholdMB,
			ActualMB:     alert.ActualMB,
			NotifyStatus: alert.NotifyStatus,
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
			var cfg struct {
				URL string `json:"url"`
			}
			_ = json.Unmarshal([]byte(channel.ConfigJSON), &cfg)
			result = append(result, dto.NotificationChannelResp{
				ChannelType: channel.ChannelType,
				Enabled:     channel.Enabled,
				Configured:  strings.TrimSpace(cfg.URL) != "",
				URL:         cfg.URL,
			})
		case channelTypeTelegram:
			var cfg struct {
				BotToken string `json:"bot_token"`
				ChatID   string `json:"chat_id"`
			}
			_ = json.Unmarshal([]byte(channel.ConfigJSON), &cfg)
			result = append(result, dto.NotificationChannelResp{
				ChannelType: channel.ChannelType,
				Enabled:     channel.Enabled,
				Configured:  strings.TrimSpace(cfg.BotToken) != "" && strings.TrimSpace(cfg.ChatID) != "",
				ChatID:      cfg.ChatID,
				TokenMasked: maskToken(cfg.BotToken),
			})
		}
	}

	return result, nil
}

func (service *AlertService) UpsertWebhookChannel(ctx context.Context, req dto.UpsertWebhookChannelReq) error {
	configJSON, err := json.Marshal(map[string]interface{}{
		"url":     strings.TrimSpace(req.URL),
		"headers": req.Headers,
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

func (service *AlertService) UpsertTelegramChannel(ctx context.Context, req dto.UpsertTelegramChannelReq) error {
	configJSON, err := json.Marshal(map[string]string{
		"bot_token": strings.TrimSpace(req.BotToken),
		"chat_id":   strings.TrimSpace(req.ChatID),
	})
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

func buildAlertKey(machineID uint, periodType string, metricType string, bucketTime time.Time) string {
	return fmt.Sprintf("%d:%s:%s:%d", machineID, periodType, metricType, bucketTime.UTC().Unix())
}

func trimMessage(message string) string {
	if len(message) <= 512 {
		return message
	}

	return message[:512]
}

func maskToken(token string) string {
	if len(token) <= 8 {
		return ""
	}

	return token[:4] + "****" + token[len(token)-4:]
}
