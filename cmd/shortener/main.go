package main

import (
	"log"

	"go.uber.org/zap"

	"github.com/anatolyi0311/cupurl/internal/config"
	"github.com/anatolyi0311/cupurl/internal/config/db"
	"github.com/anatolyi0311/cupurl/internal/server"
)

var sugar zap.SugaredLogger

func main() {
	// logging.
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

	pgdb, _ := db.InitPostgresDB(cfg, sugar)
	// if err != nil {
	// 	sugar.Fatalln(err)
	// }

	s, err := server.NewServer(cfg, sugar, pgdb)
	if err != nil {
		sugar.Fatalln(err)
	}
	s.Run()
}
