package storage

import (
	"fmt"
	"log"
	"sync"

	"github.com/trunov/go-shortener/internal/app/file"
)

type keysAndLinks map[string]string

type Storage struct {
	keysAndLinks keysAndLinks
	mtx          sync.RWMutex
	fileName     string
}

type Storager interface {
	Get(id string) (string, error)
	Add(key, link string)
	GetAll() keysAndLinks
}

func (s *Storage) Get(key string) (string, error) {
	s.mtx.RLock()
	defer s.mtx.RLock()
	v, ok := s.keysAndLinks[key]

	if !ok {
		return "", fmt.Errorf("value %s not found", key)
	}

	return v, nil
}

func NewStorage(keysAndLinks keysAndLinks, fileName string) *Storage {
	return &Storage{keysAndLinks: keysAndLinks, fileName: fileName}
}

func (s *Storage) add(key, link string) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.keysAndLinks[key] = link
}

func (s *Storage) Add(key, link string) {
	s.add(key, link)

	if s.fileName != "" {
		p, err := file.NewWriter(s.fileName)
		if err != nil {
			log.Println(err)
		}
		defer p.Close()
		p.WriteKeyAndLink(key, link)
	}
}

func (s *Storage) GetAll() keysAndLinks {
	return s.keysAndLinks
}
