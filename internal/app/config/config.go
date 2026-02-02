package config

import (
	"flag"

	"github.com/caarlos0/env"
	"github.com/sirupsen/logrus"
)

type ENVConfig struct {
	EnvServAdr     string `env:"SERVER_ADDRESS"`
	EnvBaseURL     string `env:"BASE_URL"`
	EnvStoragePath string `env:"FILE_STORAGE_PATH"`
	EnvLogLevel    string `env:"LOG_LEVEL"`
	EnvDataBase    string `env:"DATABASE_DSN"`
	EnvSecretKey   string `env:"SECRET_KEY"`
}

func NewConfig() *ENVConfig {
	var cfg ENVConfig

	flag.StringVar(&cfg.EnvServAdr, "a", "localhost:8080", "HTTP server address")
	flag.StringVar(&cfg.EnvBaseURL, "b", "http://localhost:8080", "Base URL for shortened links")
	flag.StringVar(&cfg.EnvStoragePath, "f", "/tmp/short-url-db.json", "Path for saving data file")
	flag.StringVar(&cfg.EnvLogLevel, "l", "info", "Set logg level")
	flag.StringVar(&cfg.EnvDataBase, "d", "", "Set connect DB config")
	flag.StringVar(&cfg.EnvSecretKey, "s", "", "Set secret key")

	flag.Parse()

	err := env.Parse(&cfg)
	if err != nil {
		logrus.Fatal(err)
	}

	return &cfg
}
