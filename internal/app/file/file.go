package file

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
)

type KeyAndLink struct {
	Key  string `json:"key"`
	Link string `json:"link"`
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

func (c *reader) ReadLinksAndKeys(keysAndLinks map[string]string) error {
	c.scanner.Split(bufio.ScanLines)

	for c.scanner.Scan() {
		keyAndLink := KeyAndLink{}

		err := json.Unmarshal(c.scanner.Bytes(), &keyAndLink)
		if err != nil {
			return err
		}

		keysAndLinks[keyAndLink.Key] = keyAndLink.Link
	}

	return nil
}

func SeedMapWithKeysAndLinks(fileStoragePath string, keysAndLinks map[string]string) *reader {
	reader, err := NewReader(fileStoragePath)
	if err != nil {
		log.Fatal(err)
	}

	if readerErr := reader.ReadLinksAndKeys(keysAndLinks); readerErr != nil {
		log.Fatal(err)
	}

	return reader
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

func (p *Writer) WriteKeyAndLink(key string, link string) error {
	keyAndLink := KeyAndLink{Key: key, Link: link}
	return p.encoder.Encode(keyAndLink)
}
