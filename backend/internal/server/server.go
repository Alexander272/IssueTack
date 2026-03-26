package server

import (
	"context"
	"net/http"

	"github.com/Alexander272/IssueTrack/backend/internal/config"
)

type Server struct {
	httpServer *http.Server
}

func NewServer(conf *config.Config, handler http.Handler) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:           ":" + conf.Http.Port,
			Handler:        handler,
			ReadTimeout:    conf.Http.ReadTimeout,
			WriteTimeout:   conf.Http.WriteTimeout,
			MaxHeaderBytes: conf.Http.MaxHeaderMegabytes << 20,
		},
	}
}

func (s *Server) Run() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
