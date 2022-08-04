package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"time"

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

func main() {
	SetLogLevel(logLevel)
	ctx := context.TODO()
	server := NewServer()
	server.Start(ctx)

}

type JsonRequest struct {
	Alignment string `json:"alignment"`
	Format    string `json:"alignment_format"`
	Type      string `json:"alignment_type"`
}

type JsonResponse struct {
	Success  bool   `schema:"ok"`
	Image    string `json:"image" schema:"-"`
	Error    string `json:"error" schema:"error,omitempty"`
	ImageKey string `schema:"image_key,omitempty"`
}

type Handler struct {
	response JsonResponse
	w        http.ResponseWriter
	r        *http.Request
	tmpl     *template.Template
	logger   hclog.Logger
	redis    *redis.Client
}

func NewHandler(tmpl *template.Template, w http.ResponseWriter, r *http.Request, logger hclog.Logger, redisClient *redis.Client) *Handler {
	h := &Handler{
		tmpl:   tmpl,
		w:      w,
		r:      r,
		logger: logger.With("uri", r.RequestURI),
		redis:  redisClient,
	}
	h.logger.Info(fmt.Sprintf("Incoming %s request", h.r.Method))
	return h
}

func (h *Handler) Redirect() {
	query := url.Values{}
	err := encoder.Encode(h.response, query)

	if err != nil {
		h.ReturnError(http.StatusInternalServerError, err)
		return
	}
	redirectUrl := h.r.URL
	if err != nil {
		h.ReturnError(http.StatusInternalServerError, err)
		return
	}
	redirectUrl.RawQuery = query.Encode()

	// h.logger.Debug(redirectUrl.String())
	http.Redirect(h.w, h.r, redirectUrl.String(), http.StatusFound)
}

func (h *Handler) InitialRequestLogger(logger hclog.Logger) {
	h.logger = logger.With("uri", h.r.RequestURI)
}

func (h *Handler) ReturnError(status int, err error) {
	_, filename, line, _ := runtime.Caller(1)
	errorString := err.Error()
	h.logger.With("caller", fmt.Sprintf("%s:%d", filename, line)).Error(errorString)
	h.response.Error = template.HTMLEscapeString(errorString)
	h.Redirect()
}

func (h *Handler) CheckKeyInRedis(ctx context.Context, key string) error {
	getVal, err := h.GetFromRedis(ctx, key)
	if err == nil {
		switch getVal {
		case "":
			return fmt.Errorf("error in redis with key: no value for key %s", key)
		default:
			h.logger.With("key", key).Debug("alignment image already in redis")
			return nil
		}
	}
	return err
}

func (h *Handler) StroreInRedis(ctx context.Context, key, value string) error {
	check := h.CheckKeyInRedis(ctx, key)
	if check == redis.Nil {
		h.logger.With("key", key).Debug("Storing b64 image in redis")
		return h.redis.Set(ctx, key, value, 60*time.Minute).Err()
	}
	return check
}

func (h *Handler) GetFromRedis(ctx context.Context, key string) (string, error) {
	return h.redis.Get(ctx, key).Result()
}

func (h *Handler) Index(ctx context.Context) {
	switch h.r.Method {
	case http.MethodGet:
		err := decoder.Decode(&h.response, h.r.URL.Query())
		if err != nil {
			h.ReturnError(http.StatusInternalServerError, err)
			return
		}
		if key := h.response.ImageKey; key != "" {
			val, err := h.GetFromRedis(ctx, key)
			if err != nil {
				h.ReturnError(http.StatusInternalServerError, err)
			}
			h.response.Image = val
		}

		h.tmpl.Execute(h.w, h.response)
		return
	case http.MethodPost:
		formReq, err := h.ParseForm()
		if err != nil {
			h.ReturnError(http.StatusInternalServerError, err)
			return
		}

		h.response.ImageKey = CheckSumString(formReq.Alignment)
		err = h.CheckKeyInRedis(ctx, h.response.ImageKey)

		if err == redis.Nil {
			resp, err := MakePostWithStruct(context.TODO(), fmt.Sprintf("%s/%s", pythonApiUrl, alignmentUri), formReq)
			if err != nil {
				h.ReturnError(http.StatusInternalServerError, err)
				return
			}

			defer resp.Body.Close()
			err = json.NewDecoder(resp.Body).Decode(&h.response)
			if err != nil {
				h.ReturnError(http.StatusInternalServerError, err)
				return
			}

			err = h.StroreInRedis(ctx, h.response.ImageKey, h.response.Image)

			if err != nil {
				h.ReturnError(http.StatusInternalServerError, err)
				return
			}
		} else if err != nil {
			h.ReturnError(http.StatusInternalServerError, err)
			return
		}

		h.logger.Debug("Returning result from api")

		h.response.Success = true

		h.Redirect()
	}
}

func (h *Handler) ParseForm() (JsonRequest, error) {
	h.logger.Debug("Parsing form")
	err := h.r.ParseMultipartForm(2 << 21)
	if err != nil {
		return JsonRequest{}, err
	}

	h.logger.Debug("Getting alignment from form")
	var aln string
	if file, _, err := h.r.FormFile("uploadfile"); err == nil {
		defer file.Close()
		buf := bytes.NewBuffer(nil)
		if _, err := io.Copy(buf, file); err != nil {
			return JsonRequest{}, err
		}
		aln = buf.String()
	} else {
		aln = h.r.FormValue("alignment")
	}

	if aln == "" {
		return JsonRequest{}, errors.New("no alignment provided")
	}

	h.logger.Debug("Generating request struct from form values")
	aln = base64.StdEncoding.EncodeToString([]byte(template.HTMLEscapeString(aln)))
	alnFormat := template.HTMLEscapeString(h.r.FormValue("format"))
	alnType := template.HTMLEscapeString(h.r.FormValue("type"))

	return JsonRequest{
		Alignment: aln,
		Format:    alnFormat,
		Type:      alnType,
	}, nil
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
	return resp, nil
}

func CheckSumString(str string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(str)))
}
