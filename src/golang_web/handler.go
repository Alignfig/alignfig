package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"runtime"

	"github.com/go-redis/redis/v8"
	"github.com/hashicorp/go-hclog"
)

type JsonRequest struct {
	Alignment string `json:"alignment"`
	Format    string `json:"alignment_format"`
	Type      string `json:"alignment_type"`
}

type JsonResponse struct {
	Success  bool   `schema:"ok"`
	Image    string `json:"image" schema:"-"`
	Error    string `json:"error_code" schema:"error_code,omitempty"`
	ImageKey string `json:"image_key" schema:"image_key,omitempty"`
	FetchURL string `json:"-" schema:"-"`
}

type Handler struct {
	response JsonResponse
	w        http.ResponseWriter
	r        *http.Request
	tmpl     *template.Template
	logger   hclog.Logger
	redis    *Redis
}

func NewHandler(tmpl *template.Template, w http.ResponseWriter, r *http.Request, logger hclog.Logger, redis *Redis) *Handler {
	h := &Handler{
		tmpl:   tmpl,
		w:      w,
		r:      r,
		logger: logger.With("uri", r.RequestURI),
		redis:  redis,
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

	http.Redirect(h.w, h.r, redirectUrl.String(), http.StatusFound)
}

func (h *Handler) InitialRequestLogger(logger hclog.Logger) {
	h.logger = logger.With("uri", h.r.RequestURI)
}

func (h *Handler) ReturnError(status int, err error) {
	_, filename, line, _ := runtime.Caller(1)
	errorString := err.Error()
	h.logger.With("@caller", fmt.Sprintf("%s:%d", filename, line)).Error(errorString)
	h.response.Error = template.HTMLEscapeString(errorString)
	h.Redirect()
}

func (h *Handler) Index(ctx context.Context) {
	switch h.r.Method {
	case http.MethodGet:
		err := decoder.Decode(&h.response, h.r.URL.Query())
		if err != nil {
			h.ReturnError(http.StatusInternalServerError, err)
			return
		}

		url := h.r.URL
		url.Path = fetchImageUrl
		h.response.FetchURL = url.String()
		h.tmpl.Execute(h.w, h.response)
		return
	case http.MethodPost:
		formReq, err := h.ParseForm()
		if err != nil {
			h.ReturnError(http.StatusInternalServerError, err)
			return
		}

		checkSum := formReq.Alignment + formReq.Format + formReq.Type + strconv.FormatBool(formReq.ColorSymbols) + strconv.FormatBool(formReq.LinePos) + strconv.FormatBool(formReq.Similarity)
		imageKey := CheckSumString(checkSum)
		err = h.redis.CheckKeyInRedis(ctx, imageKey)

		if err == redis.Nil {
			go h.GenerateFigFromApi(ctx, imageKey, formReq)
		} else if err != nil {
			h.ReturnError(http.StatusInternalServerError, err)
			return
		}

		h.logger.Debug("Returning result from api")

		h.response.Success = true
		h.response.ImageKey = imageKey

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
	if file, _, err := h.r.FormFile("alignment_file"); err == nil {
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
	alnFormat := template.HTMLEscapeString(h.r.FormValue("alignment_format"))
	alnType := template.HTMLEscapeString(h.r.FormValue("alignment_type"))

	return JsonRequest{
		Alignment: aln,
		Format:    alnFormat,
		Type:      alnType,
	}, nil
}

func (h *Handler) GenerateFigFromApi(ctx context.Context, imageKey string, formReq JsonRequest) {
	resp, err := MakePostWithStruct(context.TODO(), fmt.Sprintf("%s/%s", pythonApiUrl, alignmentUri), formReq)
	if err != nil {
		h.logger.Debug("error making python api request")
		return
	}

	var response JsonResponse
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		h.logger.Debug("")
		return
	}
	if response.Error != "" {
		h.logger.Debug("")
		return
	}
	err = h.redis.StroreInRedis(ctx, imageKey, response.Image)

	if err != nil {
		h.logger.Debug("")
		return
	}
}
