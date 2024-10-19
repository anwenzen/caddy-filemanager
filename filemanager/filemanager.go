package filemanager

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"go.uber.org/zap"
)

func init() {
	caddy.RegisterModule(FileManager{})
	httpcaddyfile.RegisterHandlerDirective("file_manager", parseCaddyfile)
}

// FileManager is simple manager your website,
// just only one function, delete file, no more.
type FileManager struct {
	Root string `json:"-,omitempty"`
	log  *zap.Logger
}
type message struct {
	Files []string `json:"files"`
}

func (FileManager) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.file_manager",
		New: func() caddy.Module { return new(FileManager) },
	}
}

func (fm *FileManager) Provision(ctx caddy.Context) error {
	fm.log = ctx.Logger(fm)
	return nil
}

func (fm FileManager) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	if r.URL.Path == "/api" && r.Method == http.MethodPost {
		var m message
		json.NewDecoder(r.Body).Decode(&m)
		fm.log.Debug(fmt.Sprintln("file list:", strings.Join(m.Files, " ")))
		for _, file := range m.Files {
			file = filepath.Join(fm.Root, file)
			err := os.RemoveAll(file)
			if err != nil {
				fm.log.Error(fmt.Sprintf("Remove %s fail. err: %s", file, err))
				continue
			}
			fm.log.Info(fmt.Sprintf("Remove %s success.", file))
		}
		w.Header().Set("Content-Type", "application/json")
		res := map[string]string{"code": "0", "msg": "ok"}
		json.NewEncoder(w).Encode(res)
		return nil
	}
	return next.ServeHTTP(w, r)
}

func (fm *FileManager) Validate() error {
	if fm.Root == "" {
		return fmt.Errorf("file_namager root is not uncertainty")
	}
	return nil
}
func (fm *FileManager) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	d.Next()
	for n := d.Nesting(); d.NextBlock(n); {
		if d.Val() == "root" {
			args := d.RemainingArgs()
			if len(args) != 1 {
				return d.ArgErr()
			}
			fm.Root = args[0]
			break
		} else {
			return d.ArgErr()
		}
	}
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
	_ caddy.Validator             = (*FileManager)(nil)
)
