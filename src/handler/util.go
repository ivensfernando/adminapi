package handler

import (
	"crypto/rand"
	"encoding/hex"
	"strconv"
)

func stringToFloat64(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

func generateToken() (string, error) {
	buf := make([]byte, 16)

	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return hex.EncodeToString(buf), nil
}
