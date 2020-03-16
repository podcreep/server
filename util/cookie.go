package util

import (
	"crypto/rand"
)

// CreateCookie can be used to create a random string of characters useful for being a cookie key.
func CreateCookie() (string, error) {
	var alphabet = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	bytes := make([]byte, 20)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	runes := make([]rune, 20)
	for i := range bytes {
		runes[i] = alphabet[int(bytes[i])%len(alphabet)]
	}
	return string(runes), nil
}
