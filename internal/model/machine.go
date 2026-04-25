package model

type Machine struct {
	Base
	Name             string `gorm:"column:name;size:255;not null"`
	Host             string `gorm:"column:host;size:255;not null"`
	Port             int    `gorm:"column:port;not null"`
	SSHUser          string `gorm:"column:ssh_user;size:255;not null"`
	NetworkInterface string `gorm:"column:network_interface;size:255;not null"`
	SSHKeyID         uint   `gorm:"column:ssh_key_id;not null;index"`
	CollectEnabled   bool   `gorm:"column:collect_enabled;not null"`
	Remark           string `gorm:"column:remark;type:text"`
}

func (Machine) TableName() string {
	return "machines"
}
