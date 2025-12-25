package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/anatolyi0311/cupurl/internal/model"
	"go.uber.org/zap"
)

func (s *Storage) setPsql(shortURL model.ShortURL, logger zap.SugaredLogger, userID int) (model.ShortURL, error) {
	result, err := s.db.Exec(querySetURL, shortURL.OriginalURL, shortURL.ShortURL, userID)
	if err != nil {
		return model.ShortURL{}, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return model.ShortURL{}, err
	}
	if rowsAffected == 0 {
		return model.ShortURL{ShortURL: shortURL.ShortURL, ID: shortURL.ShortURL, UserID: userID}, model.ErrURLAlreadyExists
	}
	return model.ShortURL{ShortURL: shortURL.ShortURL, ID: shortURL.ShortURL, UserID: userID}, nil
}

func (s *Storage) getPsql(hash string, _ zap.SugaredLogger, userID int) (model.ShortURL, error) {
	var originalURL string
	var isDeleted bool
	err := s.db.QueryRow(queryGetURL, hash).Scan(&originalURL, &isDeleted)
	if isDeleted {
		return model.ShortURL{OriginalURL: "", ShortURL: ""}, model.ErrDeletedURL
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.ShortURL{OriginalURL: "", ShortURL: ""}, fmt.Errorf("URL not found")
		}
		return model.ShortURL{OriginalURL: "", ShortURL: ""}, fmt.Errorf("database error: %w", err)
	}
	return model.ShortURL{OriginalURL: originalURL, ShortURL: hash, ID: hash, UserID: userID}, nil
}

func (s *Storage) setArrayPsql(request []model.SetArrayURLRequest, logger zap.SugaredLogger, userID int) ([]model.ShortURL, error) {
	tx, err := s.db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	resp := []model.ShortURL{}
	for _, item := range request {
		_, err := tx.Exec(querySetURL, item.OriginalURL, item.ShortURL, userID)
		if err != nil {
			return nil, err
		}
		resp = append(resp, model.ShortURL{
			ID:          item.ID,
			ShortURL:    s.FormatURL(item.ShortURL),
			OriginalURL: item.OriginalURL,
			UserID:      userID,
			// DeletedFlag: false,
		})
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *Storage) getArrayPsql(_ zap.SugaredLogger) ([]model.GetArrayURLResponse, error) {
	var res []model.GetArrayURLResponse
	rows, err := s.db.Query(queryGetArrayURL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	// пробегаем по всем записям
	for rows.Next() {
		var v model.GetArrayURLResponse
		err = rows.Scan(&v.Original, &v.Short)
		if err != nil {
			return nil, err
		}
		shortURL := s.cfg.Opts.BaseURL + "/" + v.Short
		v.Short = shortURL

		res = append(res, v)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return res, nil
}
func (s *Storage) DeleteDB(hash string, userID int) error {
	result, err := s.db.Exec(queryDeleteURL, hash, userID)
	if err != nil {
		return fmt.Errorf("failed delete %s; %w", hash, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed delete %s; %w", hash, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("failed delete %s; rows affected == 0", hash)
	}
	return nil
}

func (s *Storage) DeleteUrls(_ context.Context, urls []model.ShortURL, logger zap.SugaredLogger, userID int) error {
	logger.Info("db.DeleteUrls.userID: ", userID, "...", s.cfg.Opts.User)

	if len(urls) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, short := range urls {
		_, err := tx.Exec(`update cupurl set deletedFlag = true where shortURL = $1 AND userID = $2`, short.ShortURL, userID)
		if err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (s *Storage) GetUsersAndUrlsCount(ctx context.Context) (int, int, error) {
	var urlsCount int
	var usersCount int
	err := s.db.QueryRowContext(
		ctx,
		"select count('*'), count(distinct shortURL) from cupurl",
	).Scan(&urlsCount, &usersCount)
	return usersCount, urlsCount, err
}
