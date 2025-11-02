package config

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
	// "github.com/caarlos0/env/v11"
)

const (
	defaultAddr string = "localhost:8080"
)

type Config struct {
	Opts *Options
}

type Options struct {
	Addr    string `env:"SERVER_ADDRESS"`
	BaseURL string `env:"BASE_URL"`
}

func newOpts() (*Options, error) {
	// var opts Options
	// err := env.Parse(&opts)
	// if err == nil {
	// 	_, err := url.Parse("http://" + opts.Addr)
	// 	if err == nil {
	// 		_, err := url.Parse(opts.BaseURL)
	// 		if err == nil {
	// 			return &opts, nil
	// 		}
	// 	}
	// }

	envAddr := os.Getenv("SERVER_ADDRESS")
	envBaseURL := os.Getenv("BASE_URL")
	if envAddr != "" && envBaseURL != "" {
		_, err := url.Parse("http://" + envAddr)
		if err == nil {
			_, err := url.Parse(envBaseURL)
			if err == nil {
				// baseURL := "http://" + parsedBaseURL.Host
				// baseURL = strings.TrimSuffix(baseURL, "/")
				return &Options{
					Addr:    envAddr,
					BaseURL: envBaseURL,
				}, nil
			}

		}
	}

	var addr = flag.String("a", defaultAddr, "server host")
	var baseURL = flag.String("b", defaultAddr, "value before short URL")
	flag.Parse()

	if _, err := url.Parse("https://" + *addr); err != nil {
		return nil, fmt.Errorf("incorrect parametr `-a` %s", *addr)
	}
	if _, err := url.Parse("https://" + *baseURL); err != nil {
		return nil, fmt.Errorf("incorrect parametr `-b` %s", *baseURL)
	}

	if !strings.HasPrefix(*baseURL, "http://") && !strings.HasPrefix(*baseURL, "https://") {
		*baseURL = "http://" + *baseURL
	}
	*baseURL = strings.TrimSuffix(*baseURL, "/")
	return &Options{
		Addr:    *addr,
		BaseURL: *baseURL,
	}, nil
}

func NewConfig() (*Config, error) {
	// var cfg Options
	// err := env.Parse(&cfg)
	// if err == nil {
	// 	return &Config{
	// 		Opts: &cfg,
	// 	}, nil
	// }
	opts, err := newOpts()
	if err != nil {
		return nil, err
	}
	return &Config{
		Opts: opts,
	}, nil
}
