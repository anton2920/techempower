package server

import (
	"net/http"

	"github.com/anton2920/techempower/clean/internal/handler"
)

type Server struct {
	server *http.Server
}

func New(addr string, handler *handler.Handler) *Server {
	return &Server{
		server: &http.Server{
			Addr:    addr,
			Handler: handler,
		},
	}
}

func (s *Server) Run() error {
	return s.server.ListenAndServe()
}
