package dto

import "time"

type ListTrafficSamplesQuery struct {
	MachineID  *uint
	PeriodType string
	Page       int
	PageSize   int
}

type TrafficSampleResp struct {
	ID          uint      `json:"id"`
	MachineID   uint      `json:"machine_id"`
	PeriodType  string    `json:"period_type"`
	BucketTime  time.Time `json:"bucket_time"`
	UploadMB    float64   `json:"upload_mb"`
	DownloadMB  float64   `json:"download_mb"`
	TotalMB     float64   `json:"total_mb"`
	CollectedAt time.Time `json:"collected_at"`
}

type TrafficSampleListResp struct {
	Items []TrafficSampleResp `json:"items"`
	Total int64               `json:"total"`
}

type CleanupHistoryReq struct {
	DeleteSamples bool  `json:"delete_samples"`
	DeleteAlerts  bool  `json:"delete_alerts"`
	SamplesDays   *int  `json:"samples_days,omitempty"`
	AlertsDays    *int  `json:"alerts_days,omitempty"`
	MachineID     *uint `json:"machine_id,omitempty"`
}

type CleanupHistoryResp struct {
	DeletedSamples int64     `json:"deleted_samples"`
	DeletedAlerts  int64     `json:"deleted_alerts"`
	SamplesCutoff  time.Time `json:"samples_cutoff"`
	AlertsCutoff   time.Time `json:"alerts_cutoff"`
}

type CollectNowReq struct {
	MachineID *uint `json:"machine_id"`
}

type CollectNowMachineResp struct {
	MachineID   uint   `json:"machine_id"`
	Status      string `json:"status"`
	SampleCount int    `json:"sample_count"`
	Error       string `json:"error,omitempty"`
}

type CollectNowResp struct {
	Results []CollectNowMachineResp `json:"results"`
}
