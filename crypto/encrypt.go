package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"os"

	"syncase/config"
)

func EncryptFile(inputPath, outputPath string) error {
	// Load and derive key
	cfg := config.Load()
	key := deriveKey(cfg.EncryptionKey)

	// Read original file
	plaintext, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}

	// Create cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	// Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	// Create nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}

	// Encrypt and write
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return os.WriteFile(outputPath, ciphertext, 0644)
}
