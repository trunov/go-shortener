package memory

import (
	"context"
	"errors"
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

func (s *Storage) Get(_ context.Context, key string) (util.ShortenerGet, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	var shortener util.ShortenerGet
	v, ok := s.keysLinksUserID[key]

	if !ok {
		return shortener, fmt.Errorf("value %s not found", key)
	}

	shortener = util.ShortenerGet{OriginalURL: v.Link, IsDeleted: v.IsDeleted}
	return shortener, nil
}

func NewStorage(keysAndLinks util.KeysLinksUserID, fileName string) *Storage {
	return &Storage{keysLinksUserID: keysAndLinks, fileName: fileName}
}

func (s *Storage) add(ctx context.Context, key, link, userID string) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	_, err := s.GetShortenKey(ctx, link)

	if err == nil {
		return errors.New("found entry")
	}

	s.keysLinksUserID[key] = util.MapValue{Link: link, UserID: userID, IsDeleted: false}
	return nil
}

func (s *Storage) Add(ctx context.Context, key, link, userID string) error {
	err := s.add(ctx, key, link, userID)

	if err != nil {
		return err
	}

	if s.fileName != "" {
		p, err := file.NewWriter(s.fileName)
		if err != nil {
			log.Println(err)
		}
		defer p.Close()
		p.WriteKeyLinkUserID(key, link, userID)
	}

	return nil
}

func (s *Storage) GetShortenKey(_ context.Context, originalURL string) (string, error) {
	for k, v := range s.keysLinksUserID {
		if v.Link == originalURL {
			return k, nil
		}
	}

	return "", errors.New("not found")
}

func (s *Storage) GetAllLinksByUserID(_ context.Context, userID, baseURL string) ([]util.AllURLSResponse, error) {
	allUrls := []util.AllURLSResponse{}

	for key, value := range s.keysLinksUserID {
		if value.UserID == userID {
			allUrls = append(allUrls, util.AllURLSResponse{ShortURL: baseURL + "/" + key, OriginalURL: value.Link})
		}
	}

	return allUrls, nil
}

func (s *Storage) AddInBatch(ctx context.Context, br []util.BatchResponse, baseURL string) (string, error) {
	for _, v := range br {
		err := s.Add(ctx, v.ShortURL[len(baseURL)+1:], v.OriginalURL, v.UserID)
		if err != nil {
			return v.ShortURL, err
		}
	}

	return "", nil
}

func (s *Storage) DeleteURLS(_ context.Context, userID string, shortenURLS []string) error {
	for _, shortenURL := range shortenURLS {
		s.mtx.RLock()
		defer s.mtx.RUnlock()

		v, ok := s.keysLinksUserID[shortenURL]

		if ok && v.UserID == userID {
			v.IsDeleted = true
			s.keysLinksUserID[shortenURL] = v
		}
	}
	return nil
}
