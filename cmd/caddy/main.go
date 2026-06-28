// Command caddy builds a Caddy server binary that includes the file_manager
// module from this repository. It registers the standard Caddy modules plus
// the file manager handler, then delegates to Caddy's standard CLI.
package main

import (
	caddycmd "github.com/caddyserver/caddy/v2/cmd"

	// Register the standard Caddy modules (HTTP server, file server, etc.).
	_ "github.com/caddyserver/caddy/v2/modules/standard"

	// Register the file_manager handler module from this repository.
	_ "github.com/anwenzen/caddy-file-manager"
)

func main() {
	caddycmd.Main()
}
