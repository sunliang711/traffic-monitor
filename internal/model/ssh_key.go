package model

type SSHKey struct {
	Base
	Name                 string `gorm:"column:name;size:255;not null"`
	SourceType           string `gorm:"column:source_type;size:32;not null"`
	KeyType              string `gorm:"column:key_type;size:32;not null"`
	PublicKey            string `gorm:"column:public_key;type:text;not null"`
	PrivateKeyCiphertext string `gorm:"column:private_key_ciphertext;type:text;not null"`
	Fingerprint          string `gorm:"column:fingerprint;size:255;not null;uniqueIndex"`
}

func (SSHKey) TableName() string {
	return "ssh_keys"
}
