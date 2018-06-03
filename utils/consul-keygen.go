package utils

import (
	"crypto/rand"
	"encoding/base64"
)

func ConsulKeygen() (string, error) {
	key := make([]byte, 16)
	_, err := rand.Reader.Read(key)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}
