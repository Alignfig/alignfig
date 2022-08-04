package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/schema"
	"github.com/hashicorp/go-hclog"
)

var (
	pythonApiUrl = os.Getenv("PYTHON_API_URL")
	port         = os.Getenv("PORT")
	alignmentUri = os.Getenv("PYTHON_ALN_URI")
	templateFile = "templates/index.html"
	logLevel     = os.Getenv("LOG_LEVEL")
	local        = os.Getenv("LOCAL")
	redisAddr    = os.Getenv("REDIS_ADDR")

	decoder = schema.NewDecoder()
	encoder = schema.NewEncoder()

	redisClient *redis.Client
)

func main() {
	SetLogLevel(logLevel)
	ctx := context.TODO()
	server := NewServer()
	server.Start(ctx)

}

func SetLogLevel(logLevel string) {
	options := hclog.LoggerOptions{
		Level:             hclog.LevelFromString(logLevel),
		JSONFormat:        true,
		IncludeLocation:   false,
		DisableTime:       false,
		Color:             hclog.AutoColor,
		IndependentLevels: false,
	}
	hclog.SetDefault(hclog.New(&options))
}

func MakePostWithStruct(ctx context.Context, url string, jsonStruct interface{}) (*http.Response, error) {

	hclog.L().Debug("marshalling struct for new request")
	marshalJsonStruct, err := json.Marshal(jsonStruct)
	if err != nil {
		return nil, err
	}

	hclog.L().Debug("generating new post request to python api")
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(marshalJsonStruct))
	hclog.L().With("Request", req, "Error", err)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	hclog.L().Debug("python api request")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	hclog.L().With("Status", resp.Status, "Headers", resp.Header).Debug("")

	if resp.StatusCode > 399 {
		return nil, errors.New("python api returned error")
	}

	return resp, nil
}

func CheckSumString(str string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(str)))
}
