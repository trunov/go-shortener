package util

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"time"
)

type MapValue struct {
	Link   string
	UserID string
}

type AllURLSResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

func GenerateRandomString() string {
	rand.Seed(time.Now().UnixNano())

	possibleRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
	r := make([]rune, 8)

	for i := range r {
		r[i] = possibleRunes[rand.Intn(len(possibleRunes))]
	}

	return string(r)
}

func GenerateRandomUserID() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b), nil
}

func GenerateRandom(size int) ([]byte, error) {
	b := make([]byte, size)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func FindAllURLSByUserID(urlsMap map[string]MapValue, userID, baseURL string) []AllURLSResponse {
	allUrls := []AllURLSResponse{}

	for key, value := range urlsMap {
		if value.UserID == userID {
			allUrls = append(allUrls, AllURLSResponse{ShortURL: baseURL + "/" + key, OriginalURL: value.Link})
		}
	}

	return allUrls
}
