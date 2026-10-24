package handler

import (
	"encoding/json"
	"net/http"
)

// Healthz is a liveness probe only: it reports that the process is up and
// serving. Keep dependency checks (DB, cache) out of it, add /readyz instead.
func Healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
