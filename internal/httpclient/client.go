package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Config contains all configuration options for the HTTP client
type Config struct {
	BaseURL          string
	Timeout          time.Duration
	Headers          map[string]string
	RetryCount       int
	RetryWaitTime    time.Duration
	MaxRetryWaitTime time.Duration
	EnableLogging    bool
}

func DefaultConfig() *Config {
	return &Config{
		Timeout:          30 * time.Second,
		Headers:          map[string]string{},
		RetryCount:       3,
		RetryWaitTime:    1 * time.Second,
		MaxRetryWaitTime: 30 * time.Second,
		EnableLogging:    false,
	}
}

// Client is a wrapper around the standard http.Client with additional functionality
type Client struct {
	httpClient  *http.Client
	config      *Config
	middlewares []Middleware
}

// NewClient creates a new HTTP client with the given configuration
func NewClient(config *Config) *Client {
	if config == nil {
		return nil
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		config:      config,
		middlewares: []Middleware{},
	}
}

// WithMiddleware adds a middleware to the client
func (c *Client) WithMiddleware(middleware Middleware) *Client {
	c.middlewares = append(c.middlewares, middleware)
	return c
}

// Do executes a request with context and processes it through the middleware chain
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	// Apply client-level headers
	for key, value := range c.config.Headers {
		req.Header.Set(key, value)
	}

	// Clone the request to avoid modifying the original
	reqClone := req.Clone(ctx)

	// Create the handler chain
	handler := c.executeRequest

	// Apply middlewares in reverse order so they execute in the order they were added
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		handler = c.middlewares[i](handler)
	}

	return handler(ctx, reqClone)
}

// executeRequest is the final handler that executes the actual HTTP request
func (c *Client) executeRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error
	var retryCount int

	for {
		resp, err = c.httpClient.Do(req)
		if err != nil {
			// Don't retry if context is canceled or timed out
			if ctx.Err() != nil {
				return nil, fmt.Errorf("request failed: %w", err)
			}
		}

		// Check if we should retry
		shouldRetry := c.shouldRetry(resp, err)
		if !shouldRetry || retryCount >= c.config.RetryCount {
			break
		}

		// Close the response body to reuse the connection
		if resp != nil {
			resp.Body.Close()
		}

		// Wait before retrying
		waitTime := c.getRetryWaitTime(retryCount)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(waitTime):
		}

		// Clone the request to get a fresh body
		req = req.Clone(ctx)
		retryCount++
	}

	return resp, err
}

// shouldRetry determines if the request should be retried
func (c *Client) shouldRetry(resp *http.Response, err error) bool {
	// Network errors should be retried
	if err != nil {
		return true
	}

	// 5xx errors should be retried
	if resp.StatusCode >= 500 && resp.StatusCode < 600 {
		return true
	}

	// 429 Too Many Requests should be retried
	if resp.StatusCode == 429 {
		return true
	}

	return false
}

// getRetryWaitTime calculates the time to wait before retrying
func (c *Client) getRetryWaitTime(retryCount int) time.Duration {
	waitTime := c.config.RetryWaitTime * time.Duration(1<<uint(retryCount))
	if waitTime > c.config.MaxRetryWaitTime {
		waitTime = c.config.MaxRetryWaitTime
	}
	return waitTime
}

// NewRequest creates a new HTTP request with the given method, path, and body
func (c *Client) NewRequest(ctx context.Context, method, path string, body interface{}) (*http.Request, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("error marshaling body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	// Determine the full URL
	fullURL := path
	if c.config.BaseURL != "" && !strings.HasPrefix(path, "http") {
		// Simple path joining with careful handling of slashes
		baseURL := strings.TrimSuffix(c.config.BaseURL, "/")
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		fullURL = baseURL + path
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set the default Content-Type if we have a body
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

// Get performs a GET request
func (c *Client) Get(ctx context.Context, path string, result interface{}) error {
	req, err := c.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}

	return c.DoRequest(ctx, req, result)
}

// Post performs a POST request
func (c *Client) Post(ctx context.Context, path string, body, result interface{}) error {
	req, err := c.NewRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return err
	}

	return c.DoRequest(ctx, req, result)
}

// Put performs a PUT request
func (c *Client) Put(ctx context.Context, path string, body, result interface{}) error {
	req, err := c.NewRequest(ctx, http.MethodPut, path, body)
	if err != nil {
		return err
	}

	return c.DoRequest(ctx, req, result)
}

// Delete performs a DELETE request
func (c *Client) Delete(ctx context.Context, path string) error {
	req, err := c.NewRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}

	return c.DoRequest(ctx, req, nil)
}

// DoRequest performs an HTTP request and decodes the response
func (c *Client) DoRequest(ctx context.Context, req *http.Request, result interface{}) error {
	resp, err := c.Do(ctx, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check for error status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return NewAPIError(resp)
	}

	// If no result is expected, just return
	if result == nil {
		return nil
	}

	// Read and decode the response body
	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("error decoding response: %w", err)
	}

	return nil
}
