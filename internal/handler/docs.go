package handler

import (
	"net/http"

	"github.com/Lev1reG/levireg-core/api"
)

// Scalar loaded from a CDN; bump the pinned version deliberately.
const docsPage = `<!doctype html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>levireg-core API reference</title>
</head>
<body>
<div id="app"></div>
<script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference@1.73.1"></script>
<script>Scalar.createApiReference('#app', { url: '/openapi.yaml' })</script>
</body>
</html>
`

// Docs serves the API reference page, rendered client-side from /openapi.yaml.
func Docs(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(docsPage))
}

// OpenAPI serves the raw API spec.
func OpenAPI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	_, _ = w.Write(api.Spec)
}
