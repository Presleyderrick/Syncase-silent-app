package crypto

import (
	"encoding/base64"
	"errors"
)

// LoadKeyFromConfig decodes a base64-encoded key from the config.
func LoadKeyFromConfig(b64 string) ([]byte, error) {
	if b64 == "" {
		return nil, errors.New("encryption key is empty in config")
	}
	key, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, err
	}
	if len(key) != 32 {
		return nil, errors.New("encryption key must be 32 bytes")
	}
	return key, nil
}
