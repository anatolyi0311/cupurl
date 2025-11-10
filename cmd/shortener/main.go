package main

import (
	"log"

	"go.uber.org/zap"

	"github.com/anatolyi0311/cupurl/internal/config"
	"github.com/anatolyi0311/cupurl/internal/server"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalln(err)
	}
	// logging.
	logger, err := zap.NewDevelopment()
	if err != nil {
		// вызываем панику, если ошибка
		log.Fatal(err)
	}
	defer logger.Sync()
	sugar := cfg.Sugar
	sugar = *logger.Sugar()

	s, err := server.NewServer(cfg)
	if err != nil {
		sugar.Fatalln(err)
	}
	s.Run()
}
