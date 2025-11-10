package config

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	"go.uber.org/zap"
)

var sugar zap.SugaredLogger

type Config struct {
	Opts  *Options
	Sugar zap.SugaredLogger
}

type Options struct {
	Addr        string `env:"SERVER_ADDRESS"`
	BaseURL     string `env:"BASE_URL"`
	StorageFile string `env:"FILE_STORAGE_PATH"`
	AddrDB      string `env:"DATABASE_DSN"`
}

func NewConfig() (*Config, error) {
	opts, err := newOpts()
	if err != nil {
		return nil, err
	}
	return &Config{
		Opts:  opts,
		Sugar: sugar, // logging.
	}, nil
}

func newOpts() (*Options, error) {
	/* ... */
	optsEnv, err := newOptsEnv()
	if err != nil {
		return newOptsFlags()
	}
	return optsEnv, nil
}

func newOptsEnv() (*Options, error) {
	/* ... */
	envs := getEnvVars()
	envAddr,
		envBaseURL,
		envStorageFile,
		addrDB := envs[0], envs[1], envs[2], envs[3]

	storageFileValue := "storage.json"
	if envStorageFile != "" {
		storageFileValue = envStorageFile
	}

	if envAddr != "" && envBaseURL != "" {
		if _, err := url.Parse("http://" + envAddr); err == nil {
			if _, err := url.Parse(envBaseURL); err == nil {
				return &Options{
					Addr:        envAddr,
					BaseURL:     envBaseURL,
					StorageFile: storageFileValue,
					AddrDB:      addrDB,
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("error on environment variables")
}

func newOptsFlags() (*Options, error) {
	/* ... */
	envs := getEnvVars()
	envAddr,
		envBaseURL,
		envStorageFile,
		addrDB := envs[0], envs[1], envs[2], envs[3]

	storageFileValue := "storage.json"

	var addr = flag.String("a", "localhost:8080", "server host")
	var baseURL = flag.String("b", "localhost:8080", "value before short URL")
	var storageFile = flag.String("f", "storage.json", "file for save data")
	var connectDB = flag.String("d", "", "address to connect with db")
	flag.Parse()

	addrValue := *addr
	if envAddr != "" {
		addrValue = envAddr
	}
	baseURLValue := *baseURL
	if envBaseURL != "" {
		baseURLValue = envBaseURL
	}

	if _, err := url.Parse("https://" + addrValue); err != nil {
		return nil, fmt.Errorf("incorrect parametr `-a` %s", addrValue)
	}
	if _, err := url.Parse("https://" + baseURLValue); err != nil {
		return nil, fmt.Errorf("incorrect parametr `-b` %s", baseURLValue)
	}
	if connectDB == nil || *connectDB == "" {
		return nil, fmt.Errorf("incorrect parametr `-d` %s", *connectDB)
	}

	if !hasShemaURL(baseURLValue) {
		baseURLValue = "http://" + baseURLValue
	}
	baseURLValue = strings.TrimSuffix(baseURLValue, "/")

	if envStorageFile != "" {
		storageFileValue = envStorageFile
	} else {
		storageFileValue = *storageFile
	}

	connectDBValue := *connectDB
	if addrDB != "" {
		connectDBValue = addrDB
	}

	return &Options{
		Addr:        addrValue,
		BaseURL:     baseURLValue,
		StorageFile: storageFileValue,
		AddrDB:      connectDBValue,
	}, nil
}

func getEnvVars() []string {
	/* ... */
	return []string{
		os.Getenv("SERVER_ADDRESS"),
		os.Getenv("BASE_URL"),
		os.Getenv("FILE_STORAGE_PATH"),
		os.Getenv("FILE_STORAGE_PATH"),
	}
}

func hasShemaURL(baseURLValue string) bool {
	/* ... */
	return strings.HasPrefix(baseURLValue, "http://") ||
		strings.HasPrefix(baseURLValue, "https://")
}
