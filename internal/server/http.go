package server

import (
	"context"
	"net/http"
	"time"

	"github.com/anatolyi0311/cupurl/internal/config"
	"github.com/anatolyi0311/cupurl/internal/handler"
	"github.com/anatolyi0311/cupurl/internal/service"
)

type HTTP struct {
	server *http.Server
}

// Run starts http server.
func (s *HTTP) Run() error {
	return s.server.ListenAndServe()
}

func (s *HTTP) Shutdown() error {
	return s.server.Shutdown(context.Background())
}

func NewHTTP(config *config.Config, ipChecker service.IPCheckerInterface, service *service.Service, svr Server) (HTTP, error) {
	httpServer := &http.Server{
		Addr:              config.Opts.Addr,
		Handler:           handler.Compress(svr.route),
		ReadHeaderTimeout: 1 * time.Second,
	}
	server := &HTTP{
		server: httpServer,
	}
	return *server, nil
}
