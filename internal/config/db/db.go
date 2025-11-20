package db

import (
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	"github.com/anatolyi0311/cupurl/internal/config"
	"go.uber.org/zap"
)

func InitPostgresClient(cfg *config.Config, logger zap.SugaredLogger) (*sql.DB, error) {
	if cfg.Opts.AddrDB == "" {
		return nil, sql.ErrConnDone
	}

	options, err := parseDSN(cfg.Opts.AddrDB)
	if err != nil {
		return nil, err
	}
	opts := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=%s",
		options[0], options[1], options[2], options[3], options[4], options[5])
	database, err := sql.Open("postgres", opts)
	if err != nil {
		logger.Infow(
			"db.InitPostgresClient.1",
			"host", options[0],
			"port", options[1],
			"user", options[2],
			"dbname", options[3],
			"sslmode", options[5],
			"error1", err.Error(),
		)
		return nil, err
	}

	err = database.Ping()
	if err != nil {
		logger.Infow(
			"db.InitPostgresClient.2",
			"host", options[0],
			"port", options[1],
			"user", options[2],
			"dbname", options[3],
			"sslmode", options[5],
			"error2", err.Error(),
		)
		return nil, err
	}

	logger.Infow(
		"db.InitPostgresClient.3",
		"host", options[0],
		"port", options[1],
		"user", options[2],
		"dbname", options[3],
		"pasw", options[4],
		"sslmode", options[5],
	)

	return database, nil
}

func parseDSN(dsn string) ([6]string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return [6]string{}, fmt.Errorf("failed to parse DSN: %w", err)
	}

	queryParams := u.Query()
	sslmode := queryParams.Get("sslmode")

	// Разбираем host и port
	host := u.Hostname()
	port := u.Port()

	username := u.User.Username()
	password, _ := u.User.Password()

	dbname := strings.TrimPrefix(u.Path, "/")

	return [6]string{host, port, username, dbname, password, sslmode}, nil
}
