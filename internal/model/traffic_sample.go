package model

import "time"

type TrafficSample struct {
	Base
	MachineID   uint      `gorm:"column:machine_id;not null;uniqueIndex:idx_traffic_sample_bucket"`
	PeriodType  string    `gorm:"column:period_type;size:32;not null;uniqueIndex:idx_traffic_sample_bucket"`
	BucketTime  time.Time `gorm:"column:bucket_time;not null;uniqueIndex:idx_traffic_sample_bucket"`
	UploadMB    float64   `gorm:"column:upload_mb;not null"`
	DownloadMB  float64   `gorm:"column:download_mb;not null"`
	TotalMB     float64   `gorm:"column:total_mb;not null"`
	RawPayload  string    `gorm:"column:raw_payload;type:text;not null"`
	CollectedAt time.Time `gorm:"column:collected_at;not null"`
}

func (TrafficSample) TableName() string {
	return "traffic_samples"
}
