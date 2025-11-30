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

	opts, options, err := setOpts(cfg)
	if err != nil {
		return nil, err
	}
	cfg.Opts.User = options[2]

	database, err := sql.Open("postgres", opts)
	if err != nil {
		return nil, err
	}

	err = database.Ping()
	if err != nil {
		return nil, err
	}

	logger.Infow(
		"db.init",
		"host", options[0],
		"port", options[1],
		"user", options[2],
		"dbname", options[3],
		"sslmode", options[5],
	)

	return database, nil
}

func setOpts(cfg *config.Config) (string, []string, error) {
	options, err := parseDSN(cfg.Opts.AddrDB)
	if err != nil {
		return "", nil, err
	}
	opts := fmt.Sprintf(
		"host=%s port=%s user=%s dbname=%s password=%s sslmode=%s",
		options[0], options[1], options[2], options[3], options[4], options[5],
	)
	return opts, options[:], nil
}

func parseDSN(dsn string) ([6]string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return [6]string{}, fmt.Errorf("failed to parse DSN: %w", err)
	}

	queryParams := u.Query()

	// Разбираем host и port
	// host := u.Hostname()
	// port := u.Port()
	// username := u.User.Username()
	password, _ := u.User.Password()
	dbname := strings.TrimPrefix(u.Path, "/")
	// sslmode := queryParams.Get("sslmode")

	return [6]string{
		u.Hostname(),
		u.Port(),
		u.User.Username(),
		dbname,
		password,
		queryParams.Get("sslmode"),
	}, nil
}
