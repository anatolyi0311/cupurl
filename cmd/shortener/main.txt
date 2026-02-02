package main

import (
	"log"

	"go.uber.org/zap"

	"github.com/anatolyi0311/cupurl/internal/config"
	"github.com/anatolyi0311/cupurl/internal/config/db"
	"github.com/anatolyi0311/cupurl/internal/server"
	"github.com/anatolyi0311/cupurl/migrations"
)

var (
	buildVersion = "N/A" //nolint:gochecknoglobals
	buildDate    = "N/A" //nolint:gochecknoglobals
	buildCommit  = "N/A" //nolint:gochecknoglobals
)

func main() {

	// logging.
	var sugarLogger zap.SugaredLogger

	logger, err := zap.NewDevelopment()
	if err != nil {
		// вызываем панику, если ошибка
		log.Fatal(err)
	}
	defer logger.Sync()
	sugarLogger = *logger.Sugar()

	// configuration.
	cfg, err := config.NewConfig(sugarLogger)
	if err != nil {
		sugarLogger.Fatalln(err)
	}

	// init db.
	pgdb, err := db.InitPostgresClient(cfg, sugarLogger)
	if err != nil {
		sugarLogger.Warn(err)
	}

	// migrations make.
	sugarLogger.Info("Running migrations...")
	if err := migrations.Up(pgdb); err != nil {
		sugarLogger.Warn(err)
	}
	defer func() {
		// migrations.Down(pgdb)
		// sugarLogger.Info("Migrations down")
	}()
	sugarLogger.Info("Migrations applied successfully")

	// server running.
	svr, err := server.NewServer(cfg, sugarLogger, pgdb)
	if err != nil {
		sugarLogger.Fatalln(err)
	}
	svr.Run()
}
