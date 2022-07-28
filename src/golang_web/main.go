package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"io"
	"net/http"
	"os"

	"log"
)

var url = os.Getenv("PYTHON_API_URL")

type JsonResponse struct {
	Image string `json:"image"`
}

type AlignmentForm struct {
	Alignment string `json:"alignment"`
	Format    string `json:"alignment_format"`
	Type      string `json:"alignment_type"`
}

type Response struct {
	Success bool
	Image   string
	Error   error
}

func sendJson(ctx context.Context, jsonStruct AlignmentForm) (JsonResponse, error) {

	fmt.Println("URL:>", url)

	var jsonVar JsonResponse
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	marshalJsonStruct, err := json.Marshal(jsonStruct)
	if err != nil {
		return jsonVar, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(marshalJsonStruct))
	if err != nil {
		return jsonVar, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return jsonVar, err
	}
	defer resp.Body.Close()
	json.NewDecoder(resp.Body).Decode(&jsonVar)

	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Headers:", resp.Header)

	return jsonVar, nil
}
func main() {
	ctx := context.TODO()
	tmpl := template.Must(template.ParseFiles("index.html"))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			tmpl.Execute(w, nil)
			return
		}
		err := r.ParseMultipartForm(2 << 21)
		if err != nil {
			tmpl.Execute(w, Response{Error: err})
		}
		var aln string
		if file, _, err := r.FormFile("uploadfile"); err == nil {
			defer file.Close()
			buf := bytes.NewBuffer(nil)
			if _, err := io.Copy(buf, file); err != nil {
				tmpl.Execute(w, Response{Error: err})
			}
			aln = buf.String()
		} else {
			aln = r.FormValue("alignment")
		}

		aln = base64.StdEncoding.EncodeToString([]byte(html.EscapeString(aln)))
		alnFormat := html.EscapeString(r.FormValue("format"))
		alnType := html.EscapeString(r.FormValue("type"))
		alignment := AlignmentForm{
			Alignment: aln,
			Format:    alnFormat,
			Type:      alnType,
		}

		jsonResp, err := sendJson(ctx, alignment)
		if err != nil {
			log.Fatal(err)
		}

		tmpl.Execute(w, Response{
			Success: true,
			Image:   html.EscapeString(jsonResp.Image),
		})
	})

	http.ListenAndServe(":8090", nil)
}
