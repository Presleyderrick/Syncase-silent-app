// encrypt.go - FIXED
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"os"
)

// EncryptFile encrypts a file (renamed from EncryptFileStream for consistency)
func EncryptFile(key []byte, inputPath, outputPath string) error {
	in, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer out.Close()

	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}

	// write nonce first
	if _, err := out.Write(nonce); err != nil {
		return err
	}

	buffer := make([]byte, 1024*1024) // 1MB buffer

	for {
		n, err := in.Read(buffer)
		if n > 0 {
			encrypted := gcm.Seal(nil, nonce, buffer[:n], nil)
			if _, wErr := out.Write(encrypted); wErr != nil {
				return wErr
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	return nil
}