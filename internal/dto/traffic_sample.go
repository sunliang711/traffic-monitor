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
