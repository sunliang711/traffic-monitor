package dto

import "time"

type CreateMachineReq struct {
	Name             string `json:"name" binding:"required"`
	Host             string `json:"host" binding:"required"`
	Port             int    `json:"port" binding:"required"`
	SSHUser          string `json:"ssh_user" binding:"required"`
	NetworkInterface string `json:"network_interface" binding:"required"`
	SSHKeyID         uint   `json:"ssh_key_id" binding:"required"`
	CollectEnabled   bool   `json:"collect_enabled"`
	Remark           string `json:"remark"`
}

type UpdateMachineReq struct {
	Name             *string `json:"name"`
	Host             *string `json:"host"`
	Port             *int    `json:"port"`
	SSHUser          *string `json:"ssh_user"`
	NetworkInterface *string `json:"network_interface"`
	SSHKeyID         *uint   `json:"ssh_key_id"`
	CollectEnabled   *bool   `json:"collect_enabled"`
	Remark           *string `json:"remark"`
}

type MachineResp struct {
	ID               uint      `json:"id"`
	Name             string    `json:"name"`
	Host             string    `json:"host"`
	Port             int       `json:"port"`
	SSHUser          string    `json:"ssh_user"`
	NetworkInterface string    `json:"network_interface"`
	SSHKeyID         uint      `json:"ssh_key_id"`
	CollectEnabled   bool      `json:"collect_enabled"`
	Remark           string    `json:"remark"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type MachineConnectionTestResp struct {
	MachineID     uint   `json:"machine_id"`
	SSHReachable  bool   `json:"ssh_reachable"`
	VNStatReady   bool   `json:"vnstat_ready"`
	VNStatVersion string `json:"vnstat_version"`
	Status        string `json:"status"`
}
