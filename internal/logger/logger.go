package logger

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

func New(isProd bool) (*zap.Logger, error) {
	if isProd {
		return zap.NewProduction()
	}
	return zap.NewDevelopment()
}

// RequestLogger logs one line per request. zap has no HTTP adapter and chi's
// own logger writes to the standard log package, so the format is ours.
func RequestLogger(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			fields := []zap.Field{
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", ww.Status()),
				zap.Int("bytes", ww.BytesWritten()),
				zap.Duration("duration", time.Since(start)),
			}
			if ip := middleware.GetClientIP(r.Context()); ip != "" {
				fields = append(fields, zap.String("client_ip", ip))
			}
			if id := middleware.GetReqID(r.Context()); id != "" {
				fields = append(fields, zap.String("request_id", id))
			}

			switch status := ww.Status(); {
			case status >= http.StatusInternalServerError:
				log.Error("request", fields...)
			case status >= http.StatusBadRequest:
				log.Warn("request", fields...)
			default:
				log.Info("request", fields...)
			}
		})
	}
}
