package filemanager

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {

	if r.URL.Path == "/api" {

		type message struct {
			Path     string   `json:"path"`
			FileList []string `json:"file_list"`
		}
		var m message
		json.NewDecoder(r.Body).Decode(&m)
		fmt.Printf("%#v", m)
	}
	return nil
}
