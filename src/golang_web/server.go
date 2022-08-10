package main

import (
	"context"
	"encoding/json"
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
	redisStruct := NewRedisClient(redisClient, s.logger)

	s.template = template.Must(template.ParseFiles(templateFile))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handler := NewHandler(s.template, w, r, s.logger, redisStruct)
		handler.Index(ctx)
	})

	http.HandleFunc(fetchImageUrl, func(w http.ResponseWriter, r *http.Request) {
		logger := s.logger.With("uri", r.RequestURI)
		logger.Info(fmt.Sprintf("Incoming %s request", r.Method))

		var response JsonResponse

		GenerateError := func(err error, errString string) {
			logger.Error(err.Error())
			response.Error = "Error decoding query"
			json.NewEncoder(w).Encode(response)
		}

		err := decoder.Decode(&response, r.URL.Query())
		if err != nil {
			GenerateError(err, "Error decoding query")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		result, err := redisStruct.GetFromRedis(ctx, response.ImageKey)
		if err == redis.Nil {
			GenerateError(err, "No such key in redis")
			return
		} else if err != nil {
			s.logger.Error(err.Error())
			GenerateError(err, "Error getting image")
			return
		}
		response.Image = result
		json.NewEncoder(w).Encode(response)
	})

	s.logger.Debug("Starting web server")

	s.server.ListenAndServe()
}
