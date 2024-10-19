package filemanager

import (
	"html/template"
	"net/http"
	"path/filepath"

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
		files, err := filepath.Glob("./*")
		if err != nil {
			return err
		}

		tmpl := template.Must(template.New("index").Parse(`
            <!DOCTYPE html>
            <html>
            <head>
                <title>File Manager</title>
            </head>
            <body>
                <form method="post" action="/api">
                    {{range .}}
                        <input type="checkbox" name="files" value="{{.}}">{{.}}<br>
                    {{end}}
                    <button type="submit">Delete Selected</button>
                </form>
            </body>
            </html>
        `))

		if err := tmpl.Execute(w, files); err != nil {
			return err
		}
		return nil
	}
	if r.URL.Path == "/api" && r.Method == http.MethodPost {
		ServeHTTP(w, r, next)
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
