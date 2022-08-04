package main

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/hashicorp/go-hclog"
)

type Server struct {
	server   *http.Server
	template *template.Template
	logger   hclog.Logger
}

func NewServer() *Server {
	logger := hclog.L()
	return &Server{
		server: &http.Server{
			Addr: fmt.Sprintf(":%s", port),
			ErrorLog: logger.StandardLogger(&hclog.StandardLoggerOptions{
				InferLevels:              true,
				InferLevelsWithTimestamp: true,
			}),
		},
		logger: logger,
	}
}

func (s *Server) Close() {
	hclog.L().Info("closing metrics server ...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_ = s.server.Shutdown(ctx)
}

func (s *Server) Start(ctx context.Context) {
	if local != "" {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     "localhost:6379",
			Password: "",
			DB:       3,
		})
	} else {
		redisClient = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:    "mymaster",
			SentinelAddrs: []string{redisAddr},
			DB:            3,
		})

	}
	s.template = template.Must(template.ParseFiles(templateFile))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handler := NewHandler(s.template, w, r, s.logger, redisClient)
		handler.Index(ctx)
	})

	s.logger.Debug("Starting web server")

	s.server.ListenAndServe()
}
