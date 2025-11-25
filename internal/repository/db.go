package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/anatolyi0311/cupurl/internal/model"
	"go.uber.org/zap"
)

func (s *Storage) setPsql(url, hash string, logger zap.SugaredLogger) (string, error) {
	result, err := s.db.Exec(querySetURL, url, hash)

	if err != nil {
		return "", err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return "", err
	}

	if rowsAffected == 0 {
		return hash, model.ErrURLAlreadyExists
	}

	return hash, nil
}

func (s *Storage) getPsql(hash string) (string, error) {
	var originalURL string
	err := s.db.QueryRow(queryGetURL, hash).Scan(&originalURL)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("URL not found")
		}
		return "", fmt.Errorf("database error: %w", err)
	}

	return originalURL, nil
}

func (s *Storage) setArrayPsql(req []model.SetArrayURLRequest, logger zap.SugaredLogger) ([]model.SetArrayURLResponse, error) {
	tx, err := s.db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	resp := []model.SetArrayURLResponse{}
	for _, item := range req {
		_, err := tx.Exec(querySetURL, item.OriginalURL, item.ShortURL)
		if err != nil {
			return nil, err
		}
		resp = append(resp, model.SetArrayURLResponse{
			ID:  item.ID,
			URL: s.cfg.Opts.BaseURL + "/" + item.ShortURL,
		})
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return resp, nil
}
