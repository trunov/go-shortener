// Package file provides utilities for reading and writing KeyLinkUserID data
// to and from files.
package file

import (
	"bufio"
	"encoding/json"
	"os"

	"github.com/trunov/go-shortener/internal/app/util"
)

// KeyLinkUserID represents the data structure for a key, link and user ID.
type KeyLinkUserID struct {
	Key    string `json:"key"`
	Link   string `json:"link"`
	UserID string `json:"userID"`
}

// reader is responsible for reading KeyLinkUserID data from a file.
type reader struct {
	file    *os.File
	scanner *bufio.Scanner
}

// NewReader initializes a new reader instance for reading from the specified file.
// It returns a reader and any potential error encountered.
func NewReader(filename string) (*reader, error) {
	consumerFlag := os.O_RDONLY | os.O_CREATE

	file, err := os.OpenFile(filename, consumerFlag, 0644)
	if err != nil {
		return nil, err
	}

	return &reader{file: file, scanner: bufio.NewScanner(file)}, nil
}

// Close closes the file associated with the reader.
func (c *reader) Close() error {
	return c.file.Close()
}

// ReadLinksAndKeys reads key, link and user ID data from the file and populates
// the provided map with this data.
func (c *reader) ReadLinksAndKeys(keysAndLinks map[string]util.MapValue) error {
	c.scanner.Split(bufio.ScanLines)

	for c.scanner.Scan() {
		keyAndLink := KeyLinkUserID{}

		err := json.Unmarshal(c.scanner.Bytes(), &keyAndLink)
		if err != nil {
			return err
		}

		keysAndLinks[keyAndLink.Key] = util.MapValue{Link: keyAndLink.Link, UserID: keyAndLink.UserID}
	}

	return nil
}

// SeedMapWithKeysAndLinks reads key, link and user ID data from the file and
// seeds the provided map with this data. It returns a reader instance and
// any potential error encountered.
func SeedMapWithKeysAndLinks(fileStoragePath string, keysAndLinks map[string]util.MapValue) (*reader, error) {
	reader, err := NewReader(fileStoragePath)
	if err != nil {
		return nil, err
	}

	if readerErr := reader.ReadLinksAndKeys(keysAndLinks); readerErr != nil {
		return nil, err
	}

	return reader, nil
}

// Writer is responsible for writing KeyLinkUserID data to a file.
type Writer struct {
	file    *os.File
	encoder *json.Encoder
}

// NewWriter initializes a new Writer instance for writing to the specified file.
// It returns a Writer and any potential error encountered.
func NewWriter(filename string) (*Writer, error) {
	producerFlag := os.O_WRONLY | os.O_CREATE | os.O_APPEND

	file, err := os.OpenFile(filename, producerFlag, 0644)
	if err != nil {
		return nil, err
	}

	return &Writer{file: file, encoder: json.NewEncoder(file)}, nil
}

// Close closes the file associated with the Writer.
func (p *Writer) Close() error {
	return p.file.Close()
}

// WriteKeyLinkUserID writes a single KeyLinkUserID data to the file.
func (p *Writer) WriteKeyLinkUserID(key, link, userID string) error {
	keyLinkUserID := KeyLinkUserID{Key: key, Link: link, UserID: userID}
	return p.encoder.Encode(keyLinkUserID)
}
