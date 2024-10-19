package filemanager

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func init() {
	caddy.RegisterModule(FileManager{})
	httpcaddyfile.RegisterHandlerDirective("file_manager", parseCaddyfile)
}

type FileManager struct{}

func (FileManager) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.file_manager",
		New: func() caddy.Module { return new(FileManager) },
	}
}

func (fm *FileManager) Provision(ctx caddy.Context) error {
	return nil
}

func (fm FileManager) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	if r.URL.Path == "/api" && r.Method == http.MethodGet {
	}
	if r.URL.Path == "/api" && r.Method == http.MethodPost {
		buff, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		m := map[string]string{"code": "0", "msg": "ok", "recived": string(buff)}
		json.NewEncoder(w).Encode(m)
		fmt.Println(string(buff))
		return nil

	}
	return next.ServeHTTP(w, r)
}

func (fm *FileManager) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	return nil
}

func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var fm FileManager
	err := fm.UnmarshalCaddyfile(h.Dispenser)
	return fm, err
}

var (
	_ caddy.Provisioner           = (*FileManager)(nil)
	_ caddyhttp.MiddlewareHandler = (*FileManager)(nil)
	_ caddyfile.Unmarshaler       = (*FileManager)(nil)
)
