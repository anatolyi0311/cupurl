// Package repositories provides implementations for interacting with the data storage.
// It includes functionality to store, retrieve, and manage shortened URLs in a PostgreSQL database.
package url

import (
	"context"
	"errors"
	"fmt"

	"github.com/anatolyi0311/cupurl/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/sirupsen/logrus"
)

// CreateBDTable creates the "shorted_URL" table in the database if it doesn't already exist.
func (d *URLInDBRepo) CreateBDTable() error {
	ctx := context.Background()
	sqlQuery := `
		CREATE TABLE IF NOT EXISTS cupurl (
		"userid" integer NOT NULL,
		"shorturl" VARCHAR(250) NOT NULL,
		"originalurl" VARCHAR(4096) NOT NULL UNIQUE,
		"deletedflag" bool NOT NULL DEFAULT false
	)`
	_, err := d.DB.Exec(ctx, sqlQuery)
	if err != nil {
		logrus.Errorf("don't create table cupurl: %v", err)
		return err
	}
	logrus.Info("Successfully created table cupurl")
	return nil
}

// StoreURLInDB saves a mapping between an original URL and its shortened version in the database.
// It returns an error if the saving process fails.
func (d *URLInDBRepo) StoreURLInDB(ctx context.Context, originalURL, shortURL string) error {
	userID, ok := ctx.Value(models.UserIDKey).(uint32)
	if !ok {
		logrus.Errorf("context value is not userID: %v", userID)
	}
	const sqlQuery = `INSERT INTO cupurl (userid, originalurl, shorturl) VALUES ($1, $2, $3) ON CONFLICT (originalurl) DO NOTHING`
	_, err := d.DB.Exec(ctx, sqlQuery, userID, originalURL, shortURL)
	if err != nil {
		logrus.Error("url don't save in database ", err)
		return err
	}
	return nil
}

// StoreBatchURLInDB saves multiple URL mappings in the database in a batch operation.
// The input is a map where keys are shortened URLs and values are the corresponding original URLs.
// It returns an error if the batch saving process fails.
func (d *URLInDBRepo) StoreBatchURLInDB(ctx context.Context, batchURLtoStores map[string]string) error {
	userID, ok := ctx.Value(models.UserIDKey).(uint32)
	if !ok {
		logrus.Errorf("context value is not userID: %v", userID)
	}
	tx, err := d.DB.Begin(ctx)
	if err != nil {
		return err
	}
	const sqlQuery = `INSERT INTO cupurl (userid, originalurl, shorturl) VALUES ($1, $2, $3) ON CONFLICT (originalurl) DO NOTHING`
	_, err = tx.Prepare(ctx, "store_batch_url", sqlQuery)
	if err != nil {
		return err
	}
	for shortURL, originalURL := range batchURLtoStores {
		_, err = tx.Exec(ctx, "store_batch_url", userID, originalURL, shortURL)
		if err != nil {
			logrus.Error("url don't save in database ", err)
			tx.Rollback(ctx)
			return err
		}
	}
	return tx.Commit(ctx)
}

// GetOriginalURLFromDB retrieves the original URL corresponding to a given shortened URL from the database.
// It returns the original URL and any error encountered during the retrieval.
func (d *URLInDBRepo) GetOriginalURLFromDB(ctx context.Context, shortURL string) (string, error) {
	const selectQuery = `SELECT originalurl, deletedflag FROM cupurl WHERE shorturl = $1`
	var originalURL string
	var deletedFlag bool
	err := d.DB.QueryRow(ctx, selectQuery, shortURL).Scan(&originalURL, &deletedFlag)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("original URL not found")
		}
		logrus.Error("error querying for short URL: ", err)

		return "", fmt.Errorf("error querying for short URL: %w", err)
	}
	if deletedFlag {
		return "", models.ErrURLDeleted
	}
	return originalURL, nil
}

// GetShortURLFromDB retrieves the shortened version of a given original URL from the database.
// It returns the shortened URL and any error encountered during the retrieval.
func (d *URLInDBRepo) GetShortURLFromDB(ctx context.Context, originalURL string) (string, error) {
	const selectQuery = `SELECT shorturl FROM cupurl WHERE originalurl = $1`
	var shortURL string
	err := d.DB.QueryRow(ctx, selectQuery, originalURL).Scan(&shortURL)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("short URL not found: %w", err)
		}
		logrus.Error("error querying for original URL: ", err)

		return "", fmt.Errorf("error querying for original URL: %w", err)
	}
	return shortURL, nil
}

// GetUserURLSFromDB takes a slice of models.URL objects for a specific user from DB
func (d *URLInDBRepo) GetUserURLSFromDB(ctx context.Context) ([]models.URL, error) {
	const selectQuery = `SELECT shorturl,originalurl FROM cupurl WHERE userid = $1`
	userID, ok := ctx.Value(models.UserIDKey).(uint32)
	if !ok {
		logrus.Errorf("context value is not userID: %v", userID)
	}
	rows, err := d.DB.Query(ctx, selectQuery, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("userID not found: %w", err)
		}
		logrus.Error("error querying for user usersURLS: ", err)
		return nil, fmt.Errorf("error querying for user usersURLS: %w", err)
	}
	defer rows.Close()

	var userURLS []models.URL
	for rows.Next() {
		rowResult := models.URL{}
		if err = rows.Scan(&rowResult.ShortURL, &rowResult.OriginalURL); err != nil {
			logrus.Error(err)
		}
		userURLS = append(userURLS, rowResult)
	}
	if err = rows.Err(); err != nil {
		logrus.Error(err)
		return nil, err
	}
	return userURLS, nil
}

// GetShortBatchURLFromDB retrieves multiple shortened URLs corresponding to a batch of original URLs from the database.
// The input is a slice of URLRequest objects containing original URLs.
//
//	It returns found in database a map of original URLs to their shortened counterparts and any error encountered during the retrieval.
func (d *URLInDBRepo) GetShortBatchURLFromDB(ctx context.Context, batchURLRequests []models.URLRequest) (map[string]string, error) {
	var shortsURL = make(map[string]string, len(batchURLRequests))
	var shortURL string
	if d.DB == nil {
		fmt.Println("Repository not pool")
	}
	tx, err := d.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	const selectQuery = `SELECT shorturl FROM cupurl WHERE originalurl = $1`
	for _, request := range batchURLRequests {
		err = tx.QueryRow(ctx, selectQuery, request.OriginalURL).Scan(&shortURL)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			tx.Rollback(ctx)
			logrus.Error("error querying for original URL: ", err)
			return nil, fmt.Errorf("error querying for original URL: %w", err)
		}
		shortsURL[request.OriginalURL] = shortURL
	}

	return shortsURL, tx.Commit(ctx)
}
