package logger

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(isProd bool) (*zap.Logger, error) {
	if isProd {
		cfg := zap.NewProductionConfig()
		// Sampling drops repeats of the same message, and every request line
		// shares the message "request".
		cfg.Sampling = nil
		return cfg.Build()
	}
	return zap.NewDevelopment()
}

type ctxKey struct{}

// From returns the request-scoped logger RequestLogger stored in ctx, or a
// no-op logger outside a request.
func From(ctx context.Context) *zap.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*zap.Logger); ok {
		return l
	}
	return zap.NewNop()
}

// RequestLogger logs one line per request. zap has no HTTP adapter and chi's
// own logger writes to the standard log package, so the format is ours.
// Handlers get the same logger, tagged with request_id, through From.
func RequestLogger(log *zap.Logger) func(http.Handler) http.Handler {
	// Stack here always points at this middleware, never the error's origin.
	log = log.WithOptions(zap.AddStacktrace(zap.FatalLevel))
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			reqLog := log
			if id := middleware.GetReqID(r.Context()); id != "" {
				reqLog = log.With(zap.String("request_id", id))
			}

			next.ServeHTTP(ww, r.WithContext(context.WithValue(r.Context(), ctxKey{}, reqLog)))

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
			reqLog.Check(level(r.URL.Path, ww.Status()), "request").Write(fields...)
		})
	}
}

func level(path string, status int) zapcore.Level {
	switch {
	case status >= http.StatusInternalServerError:
		return zap.ErrorLevel
	case status >= http.StatusBadRequest:
		return zap.WarnLevel
	case path == "/healthz" || path == "/readyz":
		// Healthy probes fire constantly and say nothing.
		return zap.DebugLevel
	default:
		return zap.InfoLevel
	}
}

// Recoverer turns a panic into a 500 and logs it through zap, unlike chi's
// Recoverer which prints a plain-text stack to stderr. It must run inside
// RequestLogger so the panic carries request_id and the request line is still
// written.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			if rec == http.ErrAbortHandler {
				panic(rec) // net/http's signal to abort silently
			}
			From(r.Context()).Error("panic", zap.Any("panic", rec), zap.Stack("stack"))
			w.WriteHeader(http.StatusInternalServerError)
		}()
		next.ServeHTTP(w, r)
	})
}
