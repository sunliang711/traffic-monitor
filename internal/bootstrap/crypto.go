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
	key, err := cfg.MasterKey()
	if err != nil {
		return nil, fmt.Errorf("decode app master key: %w", err)
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

func (protector *DataProtector) Decrypt(ciphertext string) ([]byte, error) {
	rawCiphertext, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decode ciphertext: %w", err)
	}

	block, err := aes.NewCipher(protector.key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(rawCiphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce := rawCiphertext[:nonceSize]
	encryptedPayload := rawCiphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, encryptedPayload, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt ciphertext: %w", err)
	}

	return plaintext, nil
}
