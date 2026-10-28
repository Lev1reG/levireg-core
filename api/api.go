// Package api embeds the API spec so the binary serves it without files on disk.
package api

import _ "embed"

//go:embed openapi.yaml
var Spec []byte
