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

func ParseURL(addr *string, param string) error {
	_, err := url.Parse("https://" + *addr)
	if err != nil {
		return fmt.Errorf("incorrect parametr `-%s` %s", param, *addr)
	}
	return nil
}

func newOpts() (*Options, error) {
	var addr = flag.String("a", "localhost:8080", "server host")
	var baseURL = flag.String("b", "localhost:8080", "host before short URL")
	flag.Parse()

	if ParseURL(addr, "a") != nil {
		return nil, ParseURL(addr, "a")
	}
	if ParseURL(baseURL, "b") != nil {
		return nil, ParseURL(baseURL, "b")
	}

	if !strings.HasPrefix(*addr, "http://") && !strings.HasPrefix(*addr, "https://") {
		*baseURL = "http://" + *addr
	}
	*addr = strings.TrimSuffix(*addr, "/")
	*baseURL = strings.TrimSuffix(*baseURL, "/")

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
