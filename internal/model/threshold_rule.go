package model

type GlobalThresholdRule struct {
	Base
	PeriodType  string  `gorm:"column:period_type;size:32;not null;uniqueIndex:idx_global_threshold_dimension"`
	MetricType  string  `gorm:"column:metric_type;size:32;not null;uniqueIndex:idx_global_threshold_dimension"`
	ThresholdMB float64 `gorm:"column:threshold_mb;not null"`
	Enabled     bool    `gorm:"column:enabled;not null"`
}

func (GlobalThresholdRule) TableName() string {
	return "global_threshold_rules"
}

type MachineThresholdRule struct {
	Base
	MachineID   uint    `gorm:"column:machine_id;not null;uniqueIndex:idx_machine_threshold_dimension"`
	PeriodType  string  `gorm:"column:period_type;size:32;not null;uniqueIndex:idx_machine_threshold_dimension"`
	MetricType  string  `gorm:"column:metric_type;size:32;not null;uniqueIndex:idx_machine_threshold_dimension"`
	ThresholdMB float64 `gorm:"column:threshold_mb;not null"`
	Enabled     bool    `gorm:"column:enabled;not null"`
}

func (MachineThresholdRule) TableName() string {
	return "machine_threshold_rules"
}
