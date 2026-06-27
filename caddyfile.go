package filemanager

import (
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

// UnmarshalCaddyfile implements caddyfile.Unmarshaler. It parses the Caddyfile
// configuration block for the file_manager directive.
//
// Syntax:
//
//	file_manager {
//	    root <path>
//	    delete_password <password>
//	}
func (fm *FileManager) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	// Consume the directive name "file_manager".
	d.Next()

	// Parse the block content.
	for d.NextBlock(0) {
		switch d.Val() {
		case "root":
			if !d.NextArg() {
				return d.ArgErr()
			}
			fm.Root = d.Val()
			if d.NextArg() {
				return d.ArgErr()
			}

		case "delete_password":
			if !d.NextArg() {
				return d.ArgErr()
			}
			fm.DeletePassword = d.Val()
			if d.NextArg() {
				return d.ArgErr()
			}

		default:
			return d.Errf("unrecognized subdirective: %s", d.Val())
		}
	}

	return nil
}

// parseCaddyfile is the helper function used to register the handler directive.
// It unmarshals the Caddyfile tokens into a FileManager handler.
func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var fm FileManager
	err := fm.UnmarshalCaddyfile(h.Dispenser)
	if err != nil {
		return nil, err
	}
	return &fm, nil
}
