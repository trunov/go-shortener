// Package util provides utility functions and types related to URL shortening and user management.
package util

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"time"
)

// KeysLinksUserID is a mapping of short URLs to their corresponding MapValue.
type KeysLinksUserID map[string]MapValue

// BatchResponse represents a batch response for URL shortening,
// which includes a correlation ID, the generated short URL, and the original URL.
type BatchResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
	OriginalURL   string `json:"-"`
	UserID        string `json:"-"`
}

// ShortenerGet represents the result of getting a shortened URL's information.
type ShortenerGet struct {
	OriginalURL string
	IsDeleted   bool
}

// MapValue encapsulates the link, associated user, and deletion status for a shortened URL.
type MapValue struct {
	Link      string
	UserID    string
	IsDeleted bool
}

// AllURLSResponse represents a response containing the shortened and original URLs.
type AllURLSResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// GenerateRandomString creates a random string of length 8 consisting of alphanumeric characters.
func GenerateRandomString() string {
	const length = 8
	rand.Seed(time.Now().UnixNano())

	possibleRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
	r := make([]rune, length)

	for i := range r {
		r[i] = possibleRunes[rand.Intn(len(possibleRunes))]
	}

	return string(r)
}

// GenerateRandomUserID produces a random base64 encoded userID.
func GenerateRandomUserID() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b), nil
}

// GenerateRandom returns a slice of random bytes of the specified size.
func GenerateRandom(size int) ([]byte, error) {
	b := make([]byte, size)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// FindAllURLSByUserID retrieves all URLs associated with the specified userID from the given URLs map.
func FindAllURLSByUserID(urlsMap map[string]MapValue, userID, baseURL string) []AllURLSResponse {
	allUrls := []AllURLSResponse{}

	for key, value := range urlsMap {
		if value.UserID == userID {
			allUrls = append(allUrls, AllURLSResponse{ShortURL: baseURL + "/" + key, OriginalURL: value.Link})
		}
	}

	return allUrls
}

// GenerateChannel splits the provided slice of shortened URLs into chunks
// and sends them to a channel in chunkSize increments
func GenerateChannel(shortenURLS []string) chan []string {
	ch := make(chan []string)
	const chunkSize = 2

	go func() {
		defer close(ch)

		for i := 0; i < len(shortenURLS); i += chunkSize {
			end := i + chunkSize

			if end > len(shortenURLS) {
				end = len(shortenURLS)
			}

			ch <- shortenURLS[i:end]
		}
	}()

	return ch
}

// DefaultIfEmpty preliminary check before stdout output for main function
func DefaultIfEmpty(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}

	return value
}
