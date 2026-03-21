package server

import (
	"context"
	"net/http"

	"github.com/anatolyi0311/cupurl/internal/config"
	"github.com/anatolyi0311/cupurl/internal/models"
	"github.com/sirupsen/logrus"
)

type Server struct {
	httpServer *http.Server
}

func NewServer(cfg *config.ENVConfig, handler http.Handler) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:    cfg.EnvServAdr,
			Handler: handler,
		},
	}
}

func (s *Server) Run(addr string) error {
	logrus.Info("Starting server on: ", addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) RunTLS(addr string) error {
	logrus.Info("Starting server with TLS on: ", addr)
	return s.httpServer.ListenAndServeTLS(models.CertPEM, models.PrivateKeyPEM)
}

func (s *Server) Stop(ctx context.Context) error {
	logrus.Info("Shutting down server...")
	return s.httpServer.Shutdown(ctx)
}
