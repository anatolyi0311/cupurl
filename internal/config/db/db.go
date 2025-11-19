package db

import (
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	"github.com/anatolyi0311/cupurl/internal/config"
	"go.uber.org/zap"
)

func InitPostgresDB(cfg *config.Config, logger zap.SugaredLogger) (*sql.DB, error) {
	// ...
	u, err := url.Parse(cfg.Opts.AddrDB)
	if err != nil {
		return nil, err
	}

	// Разбираем host и port
	// host := u.Hostname()
	// port := u.Port()
	host := u.Host

	username := u.User.Username()    // `videos`
	password, _ := u.User.Password() // `userpassword`
	// ...
	dbname := strings.TrimPrefix(u.Path, "/") // `videos`
	// ...
	queryParams := u.Query()
	sslmode := queryParams.Get("sslmode")
	if sslmode == "" {
		sslmode = "disable"
	}

	addr := strings.Split(host, ":")
	if len(addr) < 1 {
		addr = []string{"postgres", "5432"}
	}

	driverName := "postgres" // "pgx"
	dataSourceName := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		addr[0], addr[1], username, password, dbname, sslmode,
	)
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	// defer db.Close()

	logger.Infow(
		"InitDB",
		"addrDB", cfg.Opts.AddrDB,
		"pathDB", cfg.Opts.PathDB,
		"host", host,
		"dbname", dbname,
		"sslmode", sslmode,
	)

	return db, nil
}
