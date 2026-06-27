// Package filemanager provides a Caddy module that serves a web-based file manager UI
// for browsing, viewing, and deleting files on the server.
package filemanager

import (
	"embed"
	"net/http"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"go.uber.org/zap"
)

//go:embed frontend/*
var frontendFS embed.FS

// init registers the module and Caddyfile directive with Caddy.
func init() {
	caddy.RegisterModule(FileManager{})
	httpcaddyfile.RegisterHandlerDirective("file_manager", parseCaddyfile)
}

// FileManager is a Caddy HTTP handler module that provides a web-based file
// management interface. It allows users to browse directories, view file info,
// and optionally delete files (protected by password).
type FileManager struct {
	// Root is the filesystem path to serve as the root of the file manager.
	Root string `json:"root,omitempty"`

	// DeletePassword is an optional password required to delete files.
	// If empty, deletion is allowed without a password.
	DeletePassword string `json:"delete_password,omitempty"`

	logger      *zap.Logger
	fileService *FileService
	diskService *DiskService
}

// CaddyModule returns the Caddy module information.
func (FileManager) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.file_manager",
		New: func() caddy.Module { return new(FileManager) },
	}
}

// Provision sets up the FileManager module. It initializes the logger,
// file service, and disk service.
func (fm *FileManager) Provision(ctx caddy.Context) error {
	fm.logger = ctx.Logger()

	// Default root to current directory if not specified.
	if fm.Root == "" {
		fm.Root = "."
	}

	fm.fileService = NewFileService(fm.Root)
	fm.diskService = NewDiskService(fm.Root)

	fm.logger.Info("file manager provisioned",
		zap.String("root", fm.Root),
		zap.Bool("delete_password_set", fm.DeletePassword != ""),
	)

	return nil
}

// Validate ensures the module configuration is valid.
func (fm *FileManager) Validate() error {
	// Root path validation is deferred to runtime (it may not exist at config time
	// in containerized environments). The fileService handles path safety at runtime.
	return nil
}

// ServeHTTP implements caddyhttp.MiddlewareHandler. It handles all requests
// for the file manager, routing to the appropriate handler or serving static assets.
func (fm *FileManager) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	// Delegate to the router which handles API and static file serving.
	fm.serveRequest(w, r)
	return nil
}

// Interface guards to ensure FileManager implements required Caddy interfaces.
var (
	_ caddy.Module                = (*FileManager)(nil)
	_ caddy.Provisioner           = (*FileManager)(nil)
	_ caddy.Validator             = (*FileManager)(nil)
	_ caddyhttp.MiddlewareHandler = (*FileManager)(nil)
	_ caddyfile.Unmarshaler       = (*FileManager)(nil)
)
