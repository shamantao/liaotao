/*
  provider_crypto.go -- API key encryption helpers for provider credentials (SET-07).
  Responsibilities: derive/load local master key, encrypt/decrypt API keys,
  and provide backward-compatible decoding of legacy plaintext values.
*/

package bindings

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const encryptedAPIKeyPrefix = "enc:v1:"

func masterKeyPath() (string, error) {
	if p := strings.TrimSpace(os.Getenv("LIAOTAO_MASTER_KEY_FILE")); p != "" {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "liaotao", "master.key"), nil
}

func readOrCreateMasterKey() ([]byte, error) {
	if env := strings.TrimSpace(os.Getenv("LIAOTAO_MASTER_KEY")); env != "" {
		raw, err := base64.StdEncoding.DecodeString(env)
		if err == nil && len(raw) >= 32 {
			hash := sha256.Sum256(raw)
			return hash[:], nil
		}
		hash := sha256.Sum256([]byte(env))
		return hash[:], nil
	}

	path, err := masterKeyPath()
	if err != nil {
		return nil, err
	}
	if b, readErr := os.ReadFile(path); readErr == nil {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(b)))
		if err != nil || len(decoded) < 32 {
			return nil, fmt.Errorf("invalid master key file")
		}
		hash := sha256.Sum256(decoded)
		return hash[:], nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	secret := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, secret); err != nil {
		return nil, err
	}
	encoded := base64.StdEncoding.EncodeToString(secret)
	if err := os.WriteFile(path, []byte(encoded), 0o600); err != nil {
		return nil, err
	}
	hash := sha256.Sum256(secret)
	return hash[:], nil
}

func encryptAPIKeyValue(plain string) (string, error) {
	trimmed := strings.TrimSpace(plain)
	if trimmed == "" {
		return "", nil
	}
	if strings.HasPrefix(trimmed, encryptedAPIKeyPrefix) {
		return trimmed, nil
	}
	key, err := readOrCreateMasterKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	cipherText := gcm.Seal(nil, nonce, []byte(trimmed), nil)
	return encryptedAPIKeyPrefix + base64.StdEncoding.EncodeToString(nonce) + ":" + base64.StdEncoding.EncodeToString(cipherText), nil
}

func decryptAPIKeyValue(stored string) (plain string, encrypted bool, err error) {
	trimmed := strings.TrimSpace(stored)
	if trimmed == "" {
		return "", false, nil
	}
	if !strings.HasPrefix(trimmed, encryptedAPIKeyPrefix) {
		return trimmed, false, nil
	}
	raw := strings.TrimPrefix(trimmed, encryptedAPIKeyPrefix)
	parts := strings.Split(raw, ":")
	if len(parts) != 2 {
		return "", true, fmt.Errorf("invalid encrypted key format")
	}
	nonce, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return "", true, err
	}
	cipherText, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", true, err
	}
	key, err := readOrCreateMasterKey()
	if err != nil {
		return "", true, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", true, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", true, err
	}
	plainBytes, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return "", true, err
	}
	return string(plainBytes), true, nil
}
