package util

import (
	"math/rand"
	"time"
)

func GenerateRandomString() string {
	rand.Seed(time.Now().UnixNano())

	possibleRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
	r := make([]rune, 8)

	for i := range r {
		r[i] = possibleRunes[rand.Intn(len(possibleRunes))]
	}

	return string(r)
}
