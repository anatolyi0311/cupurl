package main

import (
	"log"

	"go.uber.org/zap"

	"github.com/anatolyi0311/cupurl/internal/config"
	"github.com/anatolyi0311/cupurl/internal/config/db"
	"github.com/anatolyi0311/cupurl/internal/server"
	"github.com/anatolyi0311/cupurl/migrations"
)

func main() {

	// logging.
	var sugar zap.SugaredLogger
	logger, err := zap.NewDevelopment()
	if err != nil {
		// вызываем панику, если ошибка
		log.Fatal(err)
	}
	defer logger.Sync()
	sugar = *logger.Sugar()

	// configuration.
	cfg, err := config.NewConfig(sugar)
	if err != nil {
		log.Fatalln(err)
	}

	// db.
	pgdb, _ := db.InitPostgresClient(cfg, sugar)
	// if err != nil {
	// 	sugar.Fatalln(err)
	// }

	// migrations.
	sugar.Info("Running migrations...")
	err = migrations.Up(pgdb, sugar)
	if err != nil {
		sugar.Warn(err)
	}
	defer func() {
		migrations.Down(pgdb, sugar)
		sugar.Info("Migrations down")
	}()
	sugar.Info("Migrations applied successfully")

	// server.
	s, err := server.NewServer(cfg, sugar, pgdb)
	if err != nil {
		sugar.Fatalln(err)
	}
	s.Run()
}
