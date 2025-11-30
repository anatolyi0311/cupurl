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

func (s *Storage) getPsql(hash string, logger zap.SugaredLogger) (string, error) {
	var originalURL string
	// var isDeleted bool
	err := s.db.QueryRow(queryGetURL, hash).Scan(&originalURL)
	if s.cfg.Opts.Debug {
		logger.Info("getPsql", hash, originalURL)
	}

	if err != nil {
		logger.Warn("getPsql.ERROR")
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
			URL: s.FormatURL(item.ShortURL),
		})
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *Storage) getArrayPsql(logger zap.SugaredLogger) ([]model.GetArrayURLRequest, error) {
	var res []model.GetArrayURLRequest

	rows, err := s.db.Query(queryGetArrayURL)
	if err != nil {
		return nil, err
	}
	// обязательно закрываем перед возвратом функции
	defer rows.Close()

	// пробегаем по всем записям
	for rows.Next() {
		var v model.GetArrayURLRequest
		err = rows.Scan(&v.OriginalURL, &v.ShortURL)
		if err != nil {
			return nil, err
		}
		shortURL := s.FormatURL(v.ShortURL)
		v.Hash = v.ShortURL
		v.ShortURL = shortURL

		res = append(res, v)
	}

	// проверяем на ошибки
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	// logger.Info("getArrayPsql.result: ", res)
	return res, nil
}

func (s *Storage) deleteArrayPsql(ctx context.Context, urls []string, logger zap.SugaredLogger) error {
	logger.Info("db.urls", urls, len(urls))
	if len(urls) == 0 {
		return nil
	}
	// deletedAt := time.Now()
	// urlsToDelete := make(map[string][]string)
	// for _, url := range urls {
	// 	// urlsToDelete[url.CreatedByID] = append(urlsToDelete[url.CreatedByID], url.ID)
	// 	urlsToDelete[url] = append(urlsToDelete[url], url)
	// }

	for _, short := range urls {
		if short == "" {
			continue
		}
		logger.Info("db.urls.short", short)
		conn, err := s.db.Conn(ctx)
		if err != nil {
			logger.Warn("db.err.conn: ", err.Error())
			return err
		}
		if _, err := conn.ExecContext(
			ctx,
			"update cupurl set is_deleted = $1 where shortURL = $2",
			true,
			short,
		); err != nil {
			logger.Warn("db.err.exec: ", err.Error())
			return err
		}
	}

	logger.Info("db.end.nil: ")
	return nil
}
