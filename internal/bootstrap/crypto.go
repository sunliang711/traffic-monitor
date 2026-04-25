package bootstrap

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	"traffic-monitor/internal/config"
)

type DataProtector struct {
	key []byte
}

func NewDataProtector(cfg config.SecurityConfig) (*DataProtector, error) {
	key, err := base64.StdEncoding.DecodeString(cfg.AppMasterKey)
	if err != nil {
		return nil, fmt.Errorf("decode app master key: %w", err)
	}

	if len(key) != 32 {
		return nil, fmt.Errorf("app master key must decode to 32 bytes")
	}

	return &DataProtector{key: key}, nil
}

func (protector *DataProtector) Encrypt(plaintext []byte) (string, error) {
	block, err := aes.NewCipher(protector.key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}
