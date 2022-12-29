package file

import (
	"bufio"
	"encoding/json"
	"os"

	"github.com/trunov/go-shortener/internal/app/util"
)

type KeyLinkUserID struct {
	Key    string `json:"key"`
	Link   string `json:"link"`
	UserID string `json:"userID"`
}

type reader struct {
	file    *os.File
	scanner *bufio.Scanner
}

func NewReader(filename string) (*reader, error) {
	consumerFlag := os.O_RDONLY | os.O_CREATE

	file, err := os.OpenFile(filename, consumerFlag, 0644)
	if err != nil {
		return nil, err
	}

	return &reader{file: file, scanner: bufio.NewScanner(file)}, nil
}

func (c *reader) Close() error {
	return c.file.Close()
}

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

type Writer struct {
	file    *os.File
	encoder *json.Encoder
}

func NewWriter(filename string) (*Writer, error) {
	producerFlag := os.O_WRONLY | os.O_CREATE | os.O_APPEND

	file, err := os.OpenFile(filename, producerFlag, 0644)
	if err != nil {
		return nil, err
	}

	return &Writer{file: file, encoder: json.NewEncoder(file)}, nil
}

func (p *Writer) Close() error {
	return p.file.Close()
}

func (p *Writer) WriteKeyLinkUserID(key, link, userID string) error {
	keyLinkUserID := KeyLinkUserID{Key: key, Link: link, UserID: userID}
	return p.encoder.Encode(keyLinkUserID)
}
