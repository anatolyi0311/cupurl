package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/anatolyi0311/cupurl/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
)

func (s *Storage) setPsql(shortURL model.ShortURL, _ zap.SugaredLogger, userID int) (*model.ShortURL, error) {
	result, err := s.db.Exec(`INSERT INTO cupurl (originalURL, shortURL, userID) VALUES ($1, $2, $3) ON CONFLICT (originalURL) DO NOTHING;`, shortURL.OriginalURL, shortURL.ShortURL, userID)
	if err != nil {
		return nil, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		return &model.ShortURL{ShortURL: shortURL.ShortURL, ID: shortURL.ShortURL, UserID: userID}, model.ErrURLAlreadyExists
	}
	return &model.ShortURL{ShortURL: shortURL.ShortURL, ID: shortURL.ShortURL, UserID: userID}, nil
}

func (s *Storage) getPsql(hash string, _ zap.SugaredLogger) (*model.ShortURL, error) {
	var originalURL string
	var isDeleted bool
	err := s.db.QueryRow(`SELECT originalURL, deletedFlag FROM cupurl WHERE shortURL = $1;`, hash).Scan(&originalURL, &isDeleted)
	if isDeleted {
		return nil, model.ErrDeletedURL
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("URL not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}
	return &model.ShortURL{OriginalURL: originalURL, ShortURL: hash, ID: hash}, nil
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
	rows, err := s.db.Query(`SELECT originalURL, shortURL, deletedFlag FROM cupurl;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	// пробегаем по всем записям
	for rows.Next() {
		var v model.GetArrayURLResponse
		err = rows.Scan(&v.Original, &v.Short, &v.DeletedFlag)
		if v.DeletedFlag {
			return nil, model.ErrDeletedURL
		}
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

	err := s.db.QueryRow(`SELECT originalURL, deletedFlag FROM cupurl WHERE shortURL = $1 AND userID = $2;`, hash, userID).Scan(&originalURL, &isDeleted)
	if err != nil {
		logger.Warnf("%s", err)
		return fmt.Errorf("%s", sql.ErrNoRows)
	}

	result, err := s.db.Exec(`UPDATE cupurl SET deletedFlag = true WHERE shortURL = $1 AND userID = $2;`, hash, userID)
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

func (s *Storage) MarkURLsAsDeleted(ctx context.Context, URLSToDel []string) error {
	if len(URLSToDel) == 0 {
		return nil
	}
	userID, ok := ctx.Value(model.UserIDKey).(uint32)
	if !ok {
		logrus.Errorf("context value is not userID: %v", userID)
		return fmt.Errorf("invalid user context")
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		logrus.Error("Failed to begin transaction: ", err)
		return err
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				logrus.Errorf("Failed to rollback transaction: %v", rollbackErr)
			}
		}
	}()

	const sqlQuery = `UPDATE shortedurl SET deletedflag = true WHERE shorturl = ANY($1) AND userid = $2`
	_, err = tx.Exec(sqlQuery, URLSToDel, userID)
	if err != nil {
		logrus.Error("Failed to mark URLs as deleted: ", err)
		return err
	}
	logrus.Infof("Complete mark URLs as deleted: %s, %d", URLSToDel, userID)
	return tx.Commit()
}

func (s *Storage) DelUserURLS(c *gin.Context) {
	ctx := c.Request.Context()
	var URLSToDel []string
	if err := c.ShouldBindJSON(&URLSToDel); err != nil {
		logrus.Error(err)
		c.Status(http.StatusBadRequest)
		return
	}
	c.Status(http.StatusAccepted)
	s.AsyncDeleteUserURLs(ctx, URLSToDel)

}
func (s *Storage) AsyncDeleteUserURLs(ctx context.Context, URLSToDel []string) {
	go func() {
		asyncCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Minute)
		defer cancel()
		if err := s.MarkURLsAsDeleted(asyncCtx, URLSToDel); err != nil {
			logrus.Error(err)
		}
	}()
}
