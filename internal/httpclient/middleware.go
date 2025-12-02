package httpclient

import (
	"context"
	"log"
	"net/http"
	"time"
)

// Handler defines a function that handles an HTTP request
type Handler func(ctx context.Context, req *http.Request) (*http.Response, error)

// Middleware defines a function that wraps an HTTP handler
type Middleware func(Handler) Handler

// LoggingMiddleware logs the request and response details
func LoggingMiddleware(enableBody bool) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, req *http.Request) (*http.Response, error) {
			start := time.Now()
			log.Printf("[HTTP] --> %s %s", req.Method, req.URL.String())

			resp, err := next(ctx, req)

			duration := time.Since(start)
			if err != nil {
				log.Printf("[HTTP] <-- %s %s (ERROR: %v) [%s]", req.Method, req.URL.String(), err, duration)
			} else {
				log.Printf("[HTTP] <-- %s %s %d [%s]", req.Method, req.URL.String(), resp.StatusCode, duration)
			}

			return resp, err
		}
	}
}

// HeaderMiddleware adds additional headers to the request
func HeaderMiddleware(headers map[string]string) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, req *http.Request) (*http.Response, error) {
			for key, value := range headers {
				req.Header.Set(key, value)
			}
			return next(ctx, req)
		}
	}
}
