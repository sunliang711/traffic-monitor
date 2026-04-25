package dto

type ThresholdRulePayload struct {
	PeriodType     string  `json:"period_type" binding:"required"`
	MetricType     string  `json:"metric_type" binding:"required"`
	ThresholdValue float64 `json:"threshold_value" binding:"required"`
	ThresholdUnit  string  `json:"threshold_unit" binding:"required"`
	Enabled        bool    `json:"enabled"`
}

type UpsertThresholdRulesReq struct {
	Rules []ThresholdRulePayload `json:"rules" binding:"required"`
}

type ThresholdRuleResp struct {
	PeriodType     string  `json:"period_type"`
	MetricType     string  `json:"metric_type"`
	ThresholdMB    float64 `json:"threshold_mb"`
	ThresholdValue float64 `json:"threshold_value"`
	ThresholdUnit  string  `json:"threshold_unit"`
	Enabled        bool    `json:"enabled"`
	Source         string  `json:"source,omitempty"`
}
