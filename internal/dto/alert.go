package dto

import "time"

type ListAlertsQuery struct {
	MachineID  *uint
	PeriodType string
	Page       int
	PageSize   int
}

type AlertResp struct {
	ID             uint       `json:"id"`
	MachineID      uint       `json:"machine_id"`
	PeriodType     string     `json:"period_type"`
	MetricType     string     `json:"metric_type"`
	BucketTime     time.Time  `json:"bucket_time"`
	ThresholdMB    float64    `json:"threshold_mb"`
	ActualMB       float64    `json:"actual_mb"`
	NotifyStatus   string     `json:"notify_status"`
	NotifiedAt     *time.Time `json:"notified_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

type AlertListResp struct {
	Items []AlertResp `json:"items"`
	Total int64       `json:"total"`
}

type UpsertWebhookChannelReq struct {
	Enabled bool              `json:"enabled"`
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
	ProxyID *uint             `json:"proxy_id"`
}

type UpsertTelegramChannelReq struct {
	Enabled  bool   `json:"enabled"`
	BotToken string `json:"bot_token"`
	ChatID   string `json:"chat_id"`
	Message  string `json:"message"`
	ProxyID  *uint  `json:"proxy_id"`
}

type NotificationChannelResp struct {
	ChannelType string            `json:"channel_type"`
	Enabled     bool              `json:"enabled"`
	Configured  bool              `json:"configured"`
	Method      string            `json:"method,omitempty"`
	URL         string            `json:"url,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Body        string            `json:"body,omitempty"`
	ChatID      string            `json:"chat_id,omitempty"`
	Message     string            `json:"message,omitempty"`
	TokenMasked string            `json:"token_masked,omitempty"`
	ProxyID     *uint             `json:"proxy_id,omitempty"`
}

type UpsertNotificationProxyReq struct {
	Name      string `json:"name"`
	ProxyType string `json:"proxy_type"`
	URL       string `json:"url"`
}

type NotificationProxyResp struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	ProxyType string    `json:"proxy_type"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TestWebhookChannelReq struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
	ProxyID *uint             `json:"proxy_id"`
}

type TestWebhookChannelResp struct {
	StatusCode      int               `json:"status_code"`
	Body            string            `json:"body"`
	RenderedURL     string            `json:"rendered_url"`
	RenderedHeaders map[string]string `json:"rendered_headers"`
	RenderedBody    string            `json:"rendered_body"`
}

type TestTelegramChannelResp struct {
	StatusCode      int    `json:"status_code"`
	Body            string `json:"body"`
	RenderedMessage string `json:"rendered_message"`
}
