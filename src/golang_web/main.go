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
	"os"

	"github.com/hashicorp/go-hclog"
)

var (
	url          = os.Getenv("PYTHON_API_ALN_URL")
	port         = os.Getenv("PORT")
	templateFile = "index.html"
)

type AlignmentForm struct {
	Alignment string `json:"alignment"`
	Format    string `json:"alignment_format"`
	Type      string `json:"alignment_type"`
}

type Status string

var Waiting Status = "Waiting..."
var Done Status = "Done!"

type JsonResponse struct {
	Success bool
	Image   string `json:"image"`
	Error   error  `json:"error"`
}

func main() {
	SetLogLevel("debug")
	ctx := context.TODO()
	tmpl := template.Must(template.ParseFiles(templateFile))
	mux := http.NewServeMux()
	mux.HandleFunc("/", AlignmentHandler(ctx, tmpl))
	server := http.Server{
		Addr: fmt.Sprintf(":%s", port),
		ErrorLog: hclog.L().StandardLogger(&hclog.StandardLoggerOptions{
			InferLevels:              true,
			InferLevelsWithTimestamp: true,
		}),
		Handler: mux,
	}

	hclog.L().Debug("Starting web server")
	server.ListenAndServe()
}

func AlignmentHandler(ctx context.Context, tmpl *template.Template) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			hclog.L().Debug("Incoming GET request")
			tmpl.Execute(w, nil)
			return
		}
		hclog.L().Debug("Parsing form")
		err := r.ParseMultipartForm(2 << 21)
		if err != nil {
			ReturnError(w, 500, err, tmpl)
			return
		}

		hclog.L().Debug("Getting alignment")
		var aln string
		if file, _, err := r.FormFile("uploadfile"); err == nil {
			defer file.Close()
			buf := bytes.NewBuffer(nil)
			if _, err := io.Copy(buf, file); err != nil {
				ReturnError(w, 500, err, tmpl)
				return
			}
			aln = buf.String()
		} else {
			aln = r.FormValue("alignment")
		}

		hclog.L().Debug("Generating request to api")
		aln = base64.StdEncoding.EncodeToString([]byte(template.HTMLEscapeString(aln)))
		alnFormat := template.HTMLEscapeString(r.FormValue("format"))
		alnType := template.HTMLEscapeString(r.FormValue("type"))
		alignment := AlignmentForm{
			Alignment: aln,
			Format:    alnFormat,
			Type:      alnType,
		}

		hclog.L().Debug("Request to api")
		jsonResp, err := sendJson(ctx, alignment)
		if err != nil {
			ReturnError(w, 503, err, tmpl)
			return
		}

		hclog.L().Debug("Returning result")
		jsonResp.Success = true
		tmpl.Execute(w, jsonResp)
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
}

func sendJson(ctx context.Context, jsonStruct AlignmentForm) (JsonResponse, error) {

	var jsonVar JsonResponse

	hclog.L().Debug("Marshalling request")
	marshalJsonStruct, err := json.Marshal(jsonStruct)
	if err != nil {
		return jsonVar, err
	}

	hclog.L().Debug("Generating new request")
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(marshalJsonStruct))
	hclog.L().With("Request", req, "Error", err)
	if err != nil {
		return jsonVar, err
	}
	req.Header.Set("Content-Type", "application/json")

	hclog.L().Debug("Make api request")
	client := &http.Client{}
	resp, err := client.Do(req)
	hclog.L().With("Status", resp.Status, "Headers", resp.Header).Debug("")
	if err != nil {
		return jsonVar, err
	}

	if resp.StatusCode > 399 {
		return jsonVar, errors.New(resp.Status)
	}
	hclog.L().Debug("Decode response")
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&jsonVar)

	return jsonVar, err
}

func ReturnError(w http.ResponseWriter, status int, err error, tmpl *template.Template) {
	hclog.L().Error(err.Error())
	w.WriteHeader(status)
	tmpl.Execute(w, JsonResponse{Error: err, Success: true})
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
