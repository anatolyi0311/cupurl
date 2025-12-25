package config

import (
	"crypto/aes"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	"go.uber.org/zap"
)

const (
	addrPgDB = "postgres:5432"
	KeySize  = 2 * aes.BlockSize //nolint:gomnd
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
	SecretKey   string `env:"SECRET_KEY"`
	HostDB      string
	PortDB      string
	User        int `env:"USER_ID"`
	// UserDB      string
	// NameDB      string
	// PaswDB      string
	// PathDB      string
	// ParamsDB    map[string]string
	Debug         bool   `env:"DEBUG"`
	TrustedSubnet string `json:"trusted_subnet"`
	EnableHTTPS   bool   `json:"enable_https"`
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

func newOpts(_ zap.SugaredLogger) (*Options, error) {
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
	if opts.SecretKey == "" {
		opts.SecretKey = "super_secret_key_for_shortener"
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
		os.Getenv("SECRET_KEY"),
		os.Getenv("DEBUG"),
	}
}

func parseEnv() (*Options, bool) {
	/* ... */
	envs := getEnvVars()
	envAddr,
		envBaseURL,
		envStorageFile,
		envAddrDB,
		secretKey,
		debug := envs[0], envs[1], envs[2], envs[3], envs[4], envs[5]

	opts := &Options{
		// StorageFile: "storage.json",
	}
	if secretKey != "" {
		opts.SecretKey = secretKey
	}
	if debug != "" && debug == "true" {
		opts.Debug = true
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

func (c *Config) parseConfigFile(configPath string) (Config, error) {
	if configPath == "" {
		return Config{}, nil
	}

	f, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, fmt.Errorf("config file not found at: %s", configPath)
		}
		return Config{}, err
	}

	configFromFile := Config{}

	err = json.Unmarshal(f, &configFromFile)
	return configFromFile, err
}

// If the environment variable exists, return it, otherwise return the fallback value.
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
