package inmemory

import (
	"fmt"
	"log"
	"sync"

	"github.com/trunov/go-shortener/internal/app/file"
	"github.com/trunov/go-shortener/internal/app/util"
)

type Storage struct {
	keysLinksUserID util.KeysLinksUserID
	mtx             sync.RWMutex
	fileName        string
}

func (s *Storage) Get(key string) (string, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	v, ok := s.keysLinksUserID[key]

	if !ok {
		return "", fmt.Errorf("value %s not found", key)
	}

	return v.Link, nil
}

func NewStorage(keysAndLinks util.KeysLinksUserID, fileName string) *Storage {
	return &Storage{keysLinksUserID: keysAndLinks, fileName: fileName}
}

func (s *Storage) add(key, link, userID string) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.keysLinksUserID[key] = util.MapValue{Link: link, UserID: userID}
}

func (s *Storage) Add(key, link, userID string) string {
	s.add(key, link, userID)

	if s.fileName != "" {
		p, err := file.NewWriter(s.fileName)
		if err != nil {
			log.Println(err)
		}
		defer p.Close()
		p.WriteKeyLinkUserID(key, link, userID)
	}

	return ""
}

func (s *Storage) GetAllLinksByUserID(userID, baseURL string) []util.AllURLSResponse {
	allUrls := []util.AllURLSResponse{}

	for key, value := range s.keysLinksUserID {
		if value.UserID == userID {
			allUrls = append(allUrls, util.AllURLSResponse{ShortURL: baseURL + "/" + key, OriginalURL: value.Link})
		}
	}

	return allUrls
}

func (s *Storage) AddInBatch(br []util.BatchResponse, baseURL string) {
	for _, v := range br {
		s.Add(v.ShortURL[len(baseURL)+1:], v.OriginalURL, v.UserID)
	}
}
