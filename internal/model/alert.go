package model

import "time"

type Alert struct {
	Base
	MachineID    uint      `gorm:"column:machine_id;not null;index"`
	PeriodType   string    `gorm:"column:period_type;size:32;not null"`
	MetricType   string    `gorm:"column:metric_type;size:32;not null"`
	BucketTime   time.Time `gorm:"column:bucket_time;not null"`
	ThresholdMB  float64   `gorm:"column:threshold_mb;not null"`
	ActualMB     float64   `gorm:"column:actual_mb;not null"`
	AlertKey     string    `gorm:"column:alert_key;size:255;not null;uniqueIndex"`
	NotifyStatus string    `gorm:"column:notify_status;size:32;not null"`
}

func (Alert) TableName() string {
	return "alerts"
}

type NotificationChannel struct {
	Base
	ChannelType string `gorm:"column:channel_type;size:32;not null;uniqueIndex"`
	Enabled     bool   `gorm:"column:enabled;not null"`
	ConfigJSON  string `gorm:"column:config_json;type:text;not null"`
}

func (NotificationChannel) TableName() string {
	return "notification_channels"
}

type NotificationProxy struct {
	Base
	Name      string `gorm:"column:name;size:100;not null"`
	ProxyType string `gorm:"column:proxy_type;size:16;not null"`
	URL       string `gorm:"column:url;type:text;not null"`
}

func (NotificationProxy) TableName() string {
	return "notification_proxies"
}

type NotificationDelivery struct {
	Base
	AlertID         uint   `gorm:"column:alert_id;not null;index"`
	ChannelType     string `gorm:"column:channel_type;size:32;not null"`
	Success         bool   `gorm:"column:success;not null"`
	ResponseExcerpt string `gorm:"column:response_excerpt;type:text"`
	ErrorMessage    string `gorm:"column:error_message;type:text"`
}

func (NotificationDelivery) TableName() string {
	return "notification_deliveries"
}
