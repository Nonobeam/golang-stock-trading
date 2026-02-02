package middleware

import (
	"net/http"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/logger"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// LoggerMiddleware logs HTTP requests with method, path, status code, and response time
func LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		rw := newResponseWriter(w)

		// Call next handler
		next.ServeHTTP(rw, r)

		// Log after response
		duration := time.Since(start)

		logEvent := logger.Info()
		if rw.statusCode >= 400 {
			logEvent = logger.Warn()
		}
		if rw.statusCode >= 500 {
			logEvent = logger.Error()
		}

		logEvent.
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", rw.statusCode).
			Dur("duration", duration).
			Str("ip", r.RemoteAddr).
			Msg("HTTP Request")
	})
}
