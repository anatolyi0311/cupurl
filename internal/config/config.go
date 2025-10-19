package config

import (
	"flag"
	"fmt"
	"net/url"
	"strings"
)

type Config struct {
	Opts *Options
}

type Options struct {
	Addr    string
	BaseURL string
}

func newOpts() (*Options, error) {
	var addr = flag.String("a", "localhost:8080", "server host")
	var baseURL = flag.String("b", "localhost:8080", "host before short URL")
	flag.Parse()

	if _, err := url.Parse("https://" + *addr); err != nil {
		return nil, fmt.Errorf("incorrect arg `-a` %s", *addr)
	}

	if _, err := url.Parse("https://" + *baseURL); err != nil {
		return nil, fmt.Errorf("incorrect arg `-b` %s", *baseURL)
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
	opts, err := newOpts()
	if err != nil {
		return nil, err
	}
	return &Config{
		Opts: opts,
	}, nil
}
