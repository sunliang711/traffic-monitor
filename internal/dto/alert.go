package dto

import "time"

type ListAlertsQuery struct {
	MachineID  *uint
	PeriodType string
	Page       int
	PageSize   int
}

type AlertResp struct {
	ID           uint      `json:"id"`
	MachineID    uint      `json:"machine_id"`
	PeriodType   string    `json:"period_type"`
	MetricType   string    `json:"metric_type"`
	BucketTime   time.Time `json:"bucket_time"`
	ThresholdMB  float64   `json:"threshold_mb"`
	ActualMB     float64   `json:"actual_mb"`
	NotifyStatus string    `json:"notify_status"`
	CreatedAt    time.Time `json:"created_at"`
}

type AlertListResp struct {
	Items []AlertResp `json:"items"`
	Total int64       `json:"total"`
}

type UpsertWebhookChannelReq struct {
	Enabled bool              `json:"enabled"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

type UpsertTelegramChannelReq struct {
	Enabled  bool   `json:"enabled"`
	BotToken string `json:"bot_token"`
	ChatID   string `json:"chat_id"`
}

type NotificationChannelResp struct {
	ChannelType string `json:"channel_type"`
	Enabled     bool   `json:"enabled"`
	Configured  bool   `json:"configured"`
	URL         string `json:"url,omitempty"`
	ChatID      string `json:"chat_id,omitempty"`
	TokenMasked string `json:"token_masked,omitempty"`
}
