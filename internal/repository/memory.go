package repository

import (
	"fmt"

	"github.com/anatolyi0311/cupurl/internal/model"
	"go.uber.org/zap"
)

func (s *Storage) getMemory(hash string, logger zap.SugaredLogger) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// if hash == "/" || hash == "" {
	// 	for key, val := range s.memoryCache {
	// 		if val != "" {
	// 			delete(s.memoryCache, key)
	// 			// logger.Infow(
	// 			// 	"Storage.getMemory.1",
	// 			// 	"hash", hash,
	// 			// 	"key", key,
	// 			// 	"url", val,
	// 			// 	"s.memoryCache", s.memoryCache,
	// 			// )
	// 			return val, nil
	// 		}
	// 	}
	// }

	url, exist := s.memoryCache[hash]
	if !exist {
		return "", fmt.Errorf("%s not found", hash)
	}

	return url, nil
}

func (s *Storage) getArrayMemory(logger zap.SugaredLogger) ([]model.GetArrayURLRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var res []model.GetArrayURLRequest
	for key, val := range s.memoryCache {
		shortURL := s.cfg.Opts.BaseURL + "/" + key
		res = append(res, model.GetArrayURLRequest{OriginalURL: val, ShortURL: shortURL})
	}

	if len(res) == 0 {
		return []model.GetArrayURLRequest{}, fmt.Errorf("%s not found", "")
	}

	logger.Info("getArrayMemory.result: ", res)
	return res, nil
}

func (s *Storage) setMemory(url, hash string, logger zap.SugaredLogger) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exist := s.memoryCache[hash]; exist {
		existingShortURL, _ := s.getMemory(hash, logger)
		return existingShortURL, model.ErrURLAlreadyExists
	}

	s.memoryCache[hash] = url

	return hash, nil
}

func (s *Storage) setArrayMemory(req []model.SetArrayURLRequest, logger zap.SugaredLogger) ([]model.SetArrayURLResponse, error) {
	resp := []model.SetArrayURLResponse{}
	for _, item := range req {
		s.setMemory(item.OriginalURL, item.ShortURL, logger)
		resp = append(resp, model.SetArrayURLResponse{
			ID:  item.ID,
			URL: item.ShortURL,
		})
	}
	return resp, nil
}
