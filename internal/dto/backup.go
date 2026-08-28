package dto

type BackupExportReq struct {
	Password           string `json:"password" binding:"required"`
	IncludeAllMachines bool   `json:"include_all_machines"`
	MachineIDs         []uint `json:"machine_ids"`
	IncludeAllSSHKeys  bool   `json:"include_all_ssh_keys"`
	SSHKeyIDs          []uint `json:"ssh_key_ids"`
}

type BackupImportReq struct {
	Password string          `json:"password" binding:"required"`
	Backup   EncryptedBackup `json:"backup" binding:"required"`
}

type EncryptedBackup struct {
	Version    int              `json:"version"`
	Encrypted  bool             `json:"encrypted"`
	Encryption BackupEncryption `json:"encryption"`
	Payload    string           `json:"payload"`
}

type BackupEncryption struct {
	Algorithm  string `json:"algorithm"`
	KDF        string `json:"kdf"`
	Iterations int    `json:"iterations"`
	Salt       string `json:"salt"`
	Nonce      string `json:"nonce"`
}

type BackupImportResp struct {
	ImportedSSHKeys              int `json:"imported_ssh_keys"`
	SkippedSSHKeys               int `json:"skipped_ssh_keys"`
	ImportedMachines             int `json:"imported_machines"`
	SkippedMachines              int `json:"skipped_machines"`
	ImportedNotificationChannels int `json:"imported_notification_channels"`
	SkippedNotificationChannels  int `json:"skipped_notification_channels"`
	ImportedNotificationProxies  int `json:"imported_notification_proxies"`
	SkippedNotificationProxies   int `json:"skipped_notification_proxies"`
	ImportedThresholdRules       int `json:"imported_threshold_rules"`
	SkippedThresholdRules        int `json:"skipped_threshold_rules"`
}
