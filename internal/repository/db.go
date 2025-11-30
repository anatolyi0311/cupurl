package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/anatolyi0311/cupurl/internal/model"
	"go.uber.org/zap"
)

func (s *Storage) setPsql(shortURL model.ShortURL, logger zap.SugaredLogger) (model.ShortURL, error) {
	key := string(s.cfg.Opts.EncryptionKey)
	logger.Info("db.set.short", " user.key:", key)
	result, err := s.db.Exec(querySetURL, shortURL.OriginalURL, shortURL.ShortURL)

	if err != nil {
		return model.ShortURL{}, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return model.ShortURL{}, err
	}

	if rowsAffected == 0 {
		return model.ShortURL{ShortURL: shortURL.ShortURL, ID: shortURL.ShortURL}, model.ErrURLAlreadyExists
	}

	return model.ShortURL{ShortURL: shortURL.ShortURL, ID: shortURL.ShortURL}, nil
}

func (s *Storage) getPsql(hash string, logger zap.SugaredLogger) (model.ShortURL, error) {
	key := string(s.cfg.Opts.EncryptionKey)
	logger.Info("db.get.url", " user.key: ", key)
	var originalURL string
	var isDeleted bool
	err := s.db.QueryRow(queryGetURL, hash).Scan(&originalURL, &isDeleted)
	if isDeleted {
		return model.ShortURL{}, errors.New("url is deleted")
	}
	if s.cfg.Opts.Debug {
		logger.Info("getPsql", hash, originalURL)
	}
	if err != nil {
		logger.Warn("getPsql.ERROR")
		if errors.Is(err, sql.ErrNoRows) {
			return model.ShortURL{}, fmt.Errorf("URL not found")
		}
		return model.ShortURL{}, fmt.Errorf("database error: %w", err)
	}

	return model.ShortURL{OriginalURL: originalURL, ShortURL: hash, ID: hash}, nil
}

func (s *Storage) setArrayPsql(req []model.SetArrayURLRequest, _ zap.SugaredLogger) ([]model.ShortURL, error) {

	tx, err := s.db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	resp := []model.ShortURL{}
	// key := string(s.cfg.Opts.EncryptionKey)
	for _, item := range req {
		_, err := tx.Exec(querySetURL, item.OriginalURL, item.ShortURL)
		if err != nil {
			return nil, err
		}
		resp = append(resp, model.ShortURL{
			ID:          item.ID,
			ShortURL:    s.FormatURL(item.ShortURL),
			OriginalURL: item.OriginalURL,
			// OriginalURL: item.OriginalURL,
			// ShortURL:    s.FormatURL(item.ShortURL),
			// DeletedFlag: false,
		})
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *Storage) getArrayPsql(logger zap.SugaredLogger) ([]model.GetArrayURLResponse, error) {
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

	logger.Info("getarray.db", res)
	return res, nil
}
func (s *Storage) DeleteDB(hash string) error {
	result, err := s.db.Exec(queryDeleteURL, hash)
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
func (s *Storage) DeleteUrls(_ context.Context, urls []model.ShortURL, logger zap.SugaredLogger) error {
	logger.Info("db.delete.urls", urls, len(urls))
	if len(urls) == 0 {
		return nil
	}
	// deletedAt := time.Now()
	// urlsToDelete := make(map[string]string)
	// for _, url := range urls {
	// 	urlsToDelete[url.ShortURL] = url.ShortURL // append(urlsToDelete[url.ShortURL], url.ShortURL)
	// }

	// ...NEW
	tx, err := s.db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, short := range urls {
		logger.Info("db.delete.urls.short: ", short.ShortURL)
		_, err := tx.Exec(`update cupurl set deletedFlag = true where shortURL = $1`, short.ShortURL)
		if err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	// ...OLD
	// for _, short := range urlsToDelete {
	// 	// if short.ShortURL == "" {
	// 	// 	continue
	// 	// }
	// 	logger.Info("db.delete.urls.short: ", short)
	// 	// conn, err := s.db.Conn(ctx)
	// 	// if err != nil {
	// 	// 	// logger.Warn("db.conn.err: ", err.Error())
	// 	// 	return err
	// 	// }
	// 	if _, err := s.db.ExecContext(
	// 		ctx,
	// 		`update cupurl set is_deleted = $1 where shortURL = $2`, // --OR shortURL = any($3)
	// 		1,
	// 		short,
	// 		// urls,
	// 	); err != nil {
	// 		// logger.Warn("db.del.exec.err: ", err.Error())
	// 		return err
	// 	}
	// }

	logger.Info("db.del.success: ")
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
