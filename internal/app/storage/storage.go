package storage

import (
	"fmt"
	"log"
	"sync"

	"github.com/trunov/go-shortener/internal/app/file"
	"github.com/trunov/go-shortener/internal/app/util"
)

type keysLinksUserID map[string]util.MapValue

type Storage struct {
	keysAndLinks keysLinksUserID
	mtx          sync.RWMutex
	fileName     string
}

type Storager interface {
	Get(id string) (string, error)
	Add(key, link, userID string)
	GetAll() keysLinksUserID
}

func (s *Storage) Get(key string) (string, error) {
	s.mtx.RLock()
	defer s.mtx.RLock()
	v, ok := s.keysAndLinks[key]

	if !ok {
		return "", fmt.Errorf("value %s not found", key)
	}

	return v.Link, nil
}

func NewStorage(keysAndLinks keysLinksUserID, fileName string) *Storage {
	return &Storage{keysAndLinks: keysAndLinks, fileName: fileName}
}

func (s *Storage) add(key, link, userID string) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.keysAndLinks[key] = util.MapValue{Link: link, UserID: userID}
}

func (s *Storage) Add(key, link, userID string) {
	s.add(key, link, userID)

	if s.fileName != "" {
		p, err := file.NewWriter(s.fileName)
		if err != nil {
			log.Println(err)
		}
		defer p.Close()
		p.WriteKeyLinkUserID(key, link, userID)
	}
}

func (s *Storage) GetAll() keysLinksUserID {
	return s.keysAndLinks
}
