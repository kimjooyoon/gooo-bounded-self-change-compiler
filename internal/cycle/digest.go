package cycle

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func DigestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func DigestValue(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return DigestBytes(data), nil
}

func digestString(value string) string {
	return DigestBytes([]byte(value))
}
