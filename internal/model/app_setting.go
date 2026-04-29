package model

import "time"

type AppSetting struct {
	SettingKey   string    `gorm:"column:setting_key;size:128;primaryKey"`
	SettingValue string    `gorm:"column:setting_value;type:text;not null"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (AppSetting) TableName() string {
	return "app_settings"
}
