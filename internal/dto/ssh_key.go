package dto

import "time"

type ImportSSHKeyReq struct {
	Name       string `json:"name" binding:"required"`
	PrivateKey string `json:"private_key" binding:"required"`
}

type GenerateSSHKeyReq struct {
	Name string `json:"name" binding:"required"`
}

type SSHKeyResp struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	SourceType  string    `json:"source_type"`
	KeyType     string    `json:"key_type"`
	PublicKey   string    `json:"public_key"`
	Fingerprint string    `json:"fingerprint"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
