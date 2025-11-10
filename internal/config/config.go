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
	envAddr,
		envBaseURL,
		envStorageFile := getEnvVars()

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
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("error on environment variables")
}

func newOptsFlags() (*Options, error) {
	/* ... */
	envAddr,
		envBaseURL,
		envStorageFile := getEnvVars()

	storageFileValue := "storage.json"

	var addr = flag.String("a", "localhost:8080", "server host")
	var baseURL = flag.String("b", "localhost:8080", "value before short URL")
	var storageFile = flag.String("f", "storage.json", "file for save data")
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

	if !hasShemaURL(baseURLValue) {
		baseURLValue = "http://" + baseURLValue
	}
	baseURLValue = strings.TrimSuffix(baseURLValue, "/")

	if envStorageFile != "" {
		storageFileValue = envStorageFile
	} else {
		storageFileValue = *storageFile
	}

	return &Options{
		Addr:        addrValue,
		BaseURL:     baseURLValue,
		StorageFile: storageFileValue,
	}, nil
}

func getEnvVars() (string, string, string) {
	/* ... */
	return os.Getenv("SERVER_ADDRESS"),
		os.Getenv("BASE_URL"),
		os.Getenv("FILE_STORAGE_PATH")
}

func hasShemaURL(baseURLValue string) bool {
	/* ... */
	return strings.HasPrefix(baseURLValue, "http://") ||
		strings.HasPrefix(baseURLValue, "https://")
}
