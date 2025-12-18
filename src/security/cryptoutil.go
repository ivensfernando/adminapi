package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"sync"
)

const (
	encryptionKeyEnv        = "EXCHANGE_CREDENTIALS_KEY"
	defaultEncryptionKeyB64 = "Pjk+k4hske5KkKtbaKSVDOgpllRl+0EI6oCAdx88XqI="
)

var (
	encryptionKey []byte
	loadKeyOnce   sync.Once
	loadKeyErr    error
)

func getEncryptionKey() ([]byte, error) {
	loadKeyOnce.Do(func() {
		keyB64 := os.Getenv(encryptionKeyEnv)
		if keyB64 == "" {
			keyB64 = defaultEncryptionKeyB64
		}

		key, err := base64.StdEncoding.DecodeString(keyB64)
		if err != nil {
			loadKeyErr = errors.New("failed to decode EXCHANGE_CREDENTIALS_KEY from base64")
			return
		}

		switch len(key) {
		case 16, 24, 32:
			encryptionKey = key
		default:
			loadKeyErr = errors.New("EXCHANGE_CREDENTIALS_KEY must decode to 16, 24, or 32 bytes")
		}
	})

	return encryptionKey, loadKeyErr
}

func EncryptString(plaintext string) (string, error) {
	key, err := getEncryptionKey()
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

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func DecryptString(ciphertext string) (string, error) {
	key, err := getEncryptionKey()
	if err != nil {
		return "", err
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
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

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, cipherBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, cipherBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

//func ResetEncryptionKeyForTests() {
//	loadKeyOnce = sync.Once{}
//	encryptionKey = nil
//	loadKeyErr = nil
//}
