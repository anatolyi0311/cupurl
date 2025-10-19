package server

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	srv "github.com/anatolyi0311/cupurl/internal/service"
)

const (
	addr = "localhost:8080"
)

type Server struct {
	route *http.ServeMux
	su    srv.CaseURL
}

func NewServer() *Server {
	server := &Server{
		route: http.NewServeMux(),
		su:    srv.NewService(),
	}
	server.router()
	return server
}

func (s *Server) router() {
	s.route.HandleFunc("/", s.SetURL)
	s.route.HandleFunc("/{id}", s.GetURL)
}

func (s *Server) Run() {
	fmt.Println("server started ...")
	if err := http.ListenAndServe(addr, s.route); err != nil {
		log.Fatalln(err)
	}
}

func (s *Server) SetURL(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(res, "method must be POST", http.StatusBadRequest)
		return
	}

	contentType := req.Header.Get("Content-Type")
	if contentType != "text/plain" {
		http.Error(res, "Content-Type must be text/plain", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(res, "cannot read body", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	hash, err := s.su.SetURL(string(body))
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(http.StatusCreated)
	res.Write([]byte("http://localhost:8080/" + hash))
}

func (s *Server) GetURL(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(res, "method must be GET", http.StatusBadRequest)
		return
	}

	hash := strings.TrimPrefix(req.URL.Path, "/")

	url, err := s.su.GetURL(hash)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	res.Header().Set("Location", url)
	res.WriteHeader(http.StatusTemporaryRedirect)
}
