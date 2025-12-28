package repository

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/anatolyi0311/cupurl/internal/model"
	"github.com/samber/lo"
	"go.uber.org/zap"
)

func (s *Storage) saveToFile(logger zap.SugaredLogger) error {
	dir := filepath.Dir(s.cfg.Opts.StorageFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(s.s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(s.cfg.Opts.StorageFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (s *Storage) loadFromFile(logger zap.SugaredLogger) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.hasFile {
		return nil
	}

	if _, err := os.Stat(s.cfg.Opts.StorageFile); os.IsNotExist(err) {
		logger.Warn("File does not exist")
		return nil
	}

	data, err := os.ReadFile(s.cfg.Opts.StorageFile)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	if len(data) == 0 {
		return nil
	}

	if err := json.Unmarshal(data, &s.s); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return nil
}

func (s *Storage) setInFile(url, hash string, logger zap.SugaredLogger) (*model.ShortURL, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := lo.Find(s.s, func(record URLRecord) bool {
		return record.OriginalURL == url
	}); exists {
		existingShortURL, _ := s.getFromFile(hash, logger)
		return existingShortURL, model.ErrURLAlreadyExists
	}

	s.s = append(s.s, URLRecord{
		UUID:        strconv.Itoa(len(s.s)),
		ShortURL:    hash,
		OriginalURL: url,
	})
	err := s.saveToFile(logger)

	return &model.ShortURL{ShortURL: hash, OriginalURL: url, ID: hash}, err
}

func (s *Storage) getFromFile(hash string, logger zap.SugaredLogger) (*model.ShortURL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if hash == "" {
		item, exist := lo.Find(s.s, func(item URLRecord) bool {
			return item.OriginalURL != ""
		})
		if !exist {
			return nil, fmt.Errorf("%s not found", hash)
		}
		return &model.ShortURL{OriginalURL: item.OriginalURL, ShortURL: hash}, nil
	}

	item, exist := lo.Find(s.s, func(item URLRecord) bool {
		return item.ShortURL == hash
	})
	if !exist {
		return nil, fmt.Errorf("%s not found", hash)
	}

	return &model.ShortURL{OriginalURL: item.OriginalURL, ShortURL: hash}, nil
}

func (s *Storage) getArrayFromFile(logger zap.SugaredLogger) ([]model.ShortURL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var res []model.ShortURL
	for _, h := range s.s {
		shortURL := s.cfg.Opts.BaseURL + "/" + h.ShortURL
		res = append(res, model.ShortURL{OriginalURL: h.OriginalURL, ShortURL: shortURL})
	}
	if len(res) == 0 {
		return []model.ShortURL{}, fmt.Errorf("%s not found", "")
	}

	logger.Info("getArrayFromFile.result: ", res)
	return res, nil
}

func (s *Storage) setArrayInFile(req []model.SetArrayURLRequest, logger zap.SugaredLogger) ([]model.ShortURL, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	resp := []model.ShortURL{}

	for _, item := range req {
		if existingRecord, exists := lo.Find(s.s, func(record URLRecord) bool {
			return record.OriginalURL == item.OriginalURL
		}); exists {
			resp = append(resp, model.ShortURL{
				ShortURL: existingRecord.ShortURL,
			})
		} else {
			newRecord := URLRecord{
				UUID:        strconv.Itoa(len(s.s)),
				ShortURL:    item.ShortURL,
				OriginalURL: item.OriginalURL,
			}
			s.s = append(s.s, newRecord)
			resp = append(resp, model.ShortURL{
				ID:       item.ID,
				ShortURL: item.ShortURL,
			})
		}
	}

	if err := s.saveToFile(logger); err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *Storage) DeleteFile(hash string, logger zap.SugaredLogger) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lo.ForEach(s.s, func(_ URLRecord, i int) {
		if s.s[i].ShortURL == hash {
			s.s[i].DeletedFlag = true
		}
	})

	return s.saveToFile(logger)
}
