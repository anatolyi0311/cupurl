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
	result, err := s.db.Exec(`INSERT INTO cupurl (originalURL, shortURL, userID) VALUES ($1, $2, $3) ON CONFLICT (originalURL) DO NOTHING;`, shortURL.OriginalURL, shortURL.ShortURL, userID)
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
	err := s.db.QueryRow(`SELECT originalURL, deletedFlag FROM cupurl WHERE shortURL = $1;`, hash).Scan(&originalURL, &isDeleted)
	if isDeleted {
		return model.ShortURL{}, model.ErrDeletedURL
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.ShortURL{}, fmt.Errorf("URL not found")
		}
		return model.ShortURL{}, fmt.Errorf("database error: %w", err)
	}
	return model.ShortURL{OriginalURL: originalURL, ShortURL: hash, ID: hash, UserID: userID}, nil
}

func (s *Storage) setArrayPsql(request []model.SetArrayURLRequest, _ zap.SugaredLogger, userID int) ([]model.ShortURL, error) {
	tx, err := s.db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	resp := []model.ShortURL{}
	for _, item := range request {
		_, err := tx.Exec(`INSERT INTO cupurl (originalURL, shortURL, userID) VALUES ($1, $2, $3) ON CONFLICT (originalURL) DO NOTHING;`, item.OriginalURL, item.ShortURL, userID)
		if err != nil {
			return nil, err
		}
		resp = append(resp, model.ShortURL{
			ID:          item.ID,
			ShortURL:    s.FormatURL(item.ShortURL),
			OriginalURL: item.OriginalURL,
			UserID:      userID,
		})
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *Storage) getArrayPsql(_ zap.SugaredLogger) ([]model.GetArrayURLResponse, error) {
	var res []model.GetArrayURLResponse
	rows, err := s.db.Query(`SELECT originalURL, shortURL FROM cupurl;`)
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
func (s *Storage) DeleteDB(hash string, userID int, logger zap.SugaredLogger) error {
	var originalURL string
	var isDeleted bool

	err := s.db.QueryRow(`SELECT originalURL, deletedFlag FROM cupurl WHERE shortURL = $1 AND deletedFlag = true;`, hash).Scan(&originalURL, &isDeleted)
	if err != nil {
		logger.Warnf("%s", err)
		return fmt.Errorf("%s", sql.ErrNoRows)
	}

	result, err := s.db.Exec(`
		UPDATE cupurl 
		SET deletedFlag = true 
		WHERE shortURL = $1 AND userID = $2;`, hash, userID)
	if err != nil {
		return fmt.Errorf("failed delete %s; %w", hash, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed delete %s; %w", hash, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("%s", sql.ErrNoRows)
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
