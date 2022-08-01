package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/schema"
	"github.com/hashicorp/go-hclog"
)

var (
	pythonApiUrl = os.Getenv("PYTHON_API_URL")
	port         = os.Getenv("PORT")
	alignmentUri = os.Getenv("PYTHON_ALN_URI")
	templateFile = "templates/index.html"
	logLevel     = os.Getenv("LOG_LEVEL")

	decoder = schema.NewDecoder()
	encoder = schema.NewEncoder()
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
	s.template = template.Must(template.ParseFiles(templateFile))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handler := NewHandler(s.template, w, r, s.logger)
		handler.Index()
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
	Alignment    string `json:"alignment"`
	Format       string `json:"alignment_format"`
	Type         string `json:"alignment_type"`
	RefreshToken string `json:"refresh_token"`
}

type JsonResponse struct {
	Success      bool   `schema:"ok"`
	Image        string `json:"image" schema:"image"`
	Error        string `json:"error" schema:"error"`
	RefreshToken string `json:"refresh_token" schema:"refresh_token"`
}

type Handler struct {
	response JsonResponse
	w        http.ResponseWriter
	r        *http.Request
	tmpl     *template.Template
	logger   hclog.Logger
}

func NewHandler(tmpl *template.Template, w http.ResponseWriter, r *http.Request, logger hclog.Logger) *Handler {
	h := &Handler{
		tmpl:   tmpl,
		w:      w,
		r:      r,
		logger: logger.With("uri", r.RequestURI),
	}
	h.logger.Info(fmt.Sprintf("Incoming %s request", h.r.Method))
	return h
}

func (h *Handler) Redirect() {
	query := url.Values{}
	err := encoder.Encode(h.response, query)

	if err != nil {
		h.ReturnError(500, err)
		return
	}
	h.r.URL.RawQuery = query.Encode()

	http.Redirect(h.w, h.r, h.r.URL.String(), http.StatusFound)
}

func (h *Handler) InitialRequestLogger(logger hclog.Logger) {
	h.logger = logger.With("uri", h.r.RequestURI)
}

func (h *Handler) ReturnError(status int, err error) {
	errorString := err.Error()
	h.logger.Error(errorString)
	h.response.Error = template.HTMLEscapeString(errorString)
	h.Redirect()
}

func (h *Handler) Index() {
	switch h.r.Method {
	case http.MethodGet:
		err := decoder.Decode(&h.response, h.r.URL.Query())
		if err != nil {
			h.ReturnError(500, err)
			return
		}
		h.tmpl.Execute(h.w, h.response)
		return
	case http.MethodPost:
		alignment, err := h.ParseForm()
		if err != nil {
			h.ReturnError(500, err)
			return
		}
		resp, err := MakePostWithStruct(context.TODO(), fmt.Sprintf("%s/%s", pythonApiUrl, alignmentUri), alignment)
		if err != nil {
			h.ReturnError(500, err)
			return
		}

		defer resp.Body.Close()
		err = json.NewDecoder(resp.Body).Decode(&h.response)
		if err != nil {
			h.ReturnError(500, err)
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
		h.ReturnError(500, err)
		return JsonRequest{}, nil
	}

	h.logger.Debug("Getting alignment from form")
	var aln string
	if file, _, err := h.r.FormFile("uploadfile"); err == nil {
		defer file.Close()
		buf := bytes.NewBuffer(nil)
		if _, err := io.Copy(buf, file); err != nil {
			h.ReturnError(500, err)
			return JsonRequest{}, nil
		}
		aln = buf.String()
	} else {
		aln = h.r.FormValue("alignment")
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
		IncludeLocation:   true,
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
