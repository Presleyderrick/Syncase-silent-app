package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"os"

	"syncase/config"
)

func DecryptFile(inputPath, outputPath string) error {
	// Load and derive key
	cfg := config.Load()
	key := deriveKey(cfg.EncryptionKey)

	// Read encrypted file
	ciphertext, err := os.ReadFile(inputPath)
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

	// Separate nonce from ciphertext
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return err
	}
	nonce := ciphertext[:nonceSize]
	cipherData := ciphertext[nonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return err
	}

	// Save decrypted file
	return os.WriteFile(outputPath, plaintext, 0644)
}
