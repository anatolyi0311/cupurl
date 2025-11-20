package config

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	"go.uber.org/zap"
)

const (
	addrPgDB = "postgres:5432"
)

type Config struct {
	Opts *Options
}

type PgDB struct {
	urlpath  *url.URL
	addr     string
	host     string
	port     string
	user     string
	password string
	dbname   string
}

type Options struct {
	Addr        string `env:"SERVER_ADDRESS"`
	BaseURL     string `env:"BASE_URL"`
	StorageFile string `env:"FILE_STORAGE_PATH"`
	AddrDB      string `env:"DATABASE_DSN"`
	HostDB      string
	PortDB      string
	// UserDB      string
	// NameDB      string
	// PaswDB      string
	// PathDB      string
	// ParamsDB    map[string]string
}

func NewConfig(logger zap.SugaredLogger) (*Config, error) {
	opts, err := newOpts(logger)
	if err != nil {
		return nil, err
	}
	return &Config{
		Opts: opts,
	}, nil
}

func newOpts(logger zap.SugaredLogger) (*Options, error) {
	/* ... */
	opts, ok := parseEnv()
	if ok {
		return opts, nil
	}
	var addr = flag.String("a", "localhost:8080", "server host")
	var baseURL = flag.String("b", "localhost:8080", "value before short URL")
	var storageFile = flag.String("f", "", "file for save data") // storage.json
	var psqlHost = flag.String("d", "", "psql data")
	flag.Parse()

	if opts.Addr == "" {
		opts.Addr = *addr
	}
	if opts.BaseURL == "" {
		opts.BaseURL = *baseURL
	}
	if opts.StorageFile == "" { // storage.json
		opts.StorageFile = *storageFile
	}
	if opts.AddrDB == "" {
		opts.AddrDB = *psqlHost
	}
	if _, err := url.Parse("https://" + opts.Addr); err != nil {
		return nil, fmt.Errorf("incorrect parametr `-a` %s", opts.Addr)
	}

	if _, err := url.Parse("https://" + opts.BaseURL); err != nil {
		return nil, fmt.Errorf("incorrect parametr `-b` %s", opts.BaseURL)
	}

	if _, err := url.Parse("https://" + opts.AddrDB); err != nil {
		return nil, fmt.Errorf("incorrect parametr `-d` %s", opts.AddrDB)
	}

	if !strings.HasPrefix(opts.BaseURL, "http://") && !strings.HasPrefix(opts.BaseURL, "https://") {
		opts.BaseURL = "http://" + opts.BaseURL
	}
	opts.BaseURL = strings.TrimSuffix(opts.BaseURL, "/")
	// logger.Infow(
	// 	"cfg",
	// 	"opts", true,
	// )
	return opts, nil
}

func getEnvVars() []string {
	/* ... */
	return []string{
		os.Getenv("SERVER_ADDRESS"),
		os.Getenv("BASE_URL"),
		os.Getenv("FILE_STORAGE_PATH"),
		os.Getenv("DATABASE_DSN"),
	}
}

func parseEnv() (*Options, bool) {
	/* ... */
	envs := getEnvVars()
	envAddr,
		envBaseURL,
		envStorageFile,
		envAddrDB := envs[0], envs[1], envs[2], envs[3]

	opts := &Options{
		// StorageFile: "storage.json",
	}
	if envStorageFile != "" {
		opts.StorageFile = envStorageFile
	}
	if _, err := url.Parse("http://" + envAddr); err == nil {
		opts.Addr = envAddr
	}
	if _, err := url.Parse(envBaseURL); err == nil {
		opts.BaseURL = envBaseURL
	}
	if _, err := url.Parse(envAddrDB); err == nil {
		opts.AddrDB = envAddrDB
	}
	return opts, (opts.Addr != "" && opts.BaseURL != "" && opts.AddrDB != "")
}
