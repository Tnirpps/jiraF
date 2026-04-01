package httpclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type TestResponse struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

// Tests that the HTTP client correctly performs GET requests with proper headers and authorization
// Verifies that the request method, path, and headers are correctly set and the response is properly decoded
func TestClient_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/test" {
			t.Errorf("Expected /test path, got %s", r.URL.Path)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type header to be set")
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Expected Authorization header to be set")
		}

		resp := TestResponse{
			Message: "Success",
			Status:  "OK",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := DefaultConfig()
	config.BaseURL = server.URL
	config.Headers = map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer test-token",
	}

	client := NewClient(config)

	var response TestResponse
	err := client.Get(context.Background(), "/test", &response)
	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}

	if response.Message != "Success" || response.Status != "OK" {
		t.Errorf("Unexpected response: %+v", response)
	}
}

// Tests that the HTTP client correctly performs POST requests with JSON body
// Verifies that the request body is properly encoded and sent, and the response is correctly decoded
func TestClient_Post(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/test" {
			t.Errorf("Expected /test path, got %s", r.URL.Path)
		}

		var requestBody map[string]string
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Errorf("Error decoding request body: %v", err)
		}
		defer r.Body.Close()

		if requestBody["key"] != "value" {
			t.Errorf("Expected request body to have key=value, got %v", requestBody)
		}

		resp := TestResponse{
			Message: "Created",
			Status:  "OK",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := DefaultConfig()
	config.BaseURL = server.URL
	config.Headers = map[string]string{
		"Content-Type": "application/json",
	}

	client := NewClient(config)

	requestBody := map[string]string{
		"key": "value",
	}
	var response TestResponse
	err := client.Post(context.Background(), "/test", requestBody, &response)
	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}

	if response.Message != "Created" || response.Status != "OK" {
		t.Errorf("Unexpected response: %+v", response)
	}
}

// Tests that middleware functions are correctly applied to HTTP requests
// Verifies that custom headers added by middleware are present in the outgoing request
func TestClient_Middleware(t *testing.T) {
	var headerValue string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headerValue = r.Header.Get("X-Test-Header")
		resp := TestResponse{
			Message: "Success",
			Status:  "OK",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := DefaultConfig()
	config.BaseURL = server.URL

	client := NewClient(config)

	client.WithMiddleware(func(next Handler) Handler {
		return func(ctx context.Context, req *http.Request) (*http.Response, error) {
			req.Header.Set("X-Test-Header", "test-value")
			return next(ctx, req)
		}
	})

	var response TestResponse
	err := client.Get(context.Background(), "/test", &response)
	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}

	if headerValue != "test-value" {
		t.Errorf("Expected X-Test-Header to be set to test-value, got %s", headerValue)
	}
}

// Tests that the HTTP client correctly retries failed requests according to configuration
// Verifies that the client retries on 5xx errors and succeeds after the configured number of attempts
func TestClient_Retry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++

		if attempts <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		resp := TestResponse{
			Message: "Success after retry",
			Status:  "OK",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := DefaultConfig()
	config.BaseURL = server.URL
	config.RetryCount = 3
	config.RetryWaitTime = 10 * time.Millisecond

	client := NewClient(config)

	var response TestResponse
	err := client.Get(context.Background(), "/test", &response)
	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}

	if response.Message != "Success after retry" || response.Status != "OK" {
		t.Errorf("Unexpected response: %+v", response)
	}
}

// Tests that the HTTP client correctly handles API errors and provides appropriate error information
// Verifies that error status codes are properly detected and helper functions (IsNotFound, IsForbidden) work correctly
func TestClient_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "Not Found", "code": 404}`))
	}))
	defer server.Close()

	config := DefaultConfig()
	config.BaseURL = server.URL
	config.RetryCount = 0

	client := NewClient(config)

	var response TestResponse
	err := client.Get(context.Background(), "/test", &response)

	if err == nil {
		t.Fatalf("Expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("Expected APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status code 404, got %d", apiErr.StatusCode)
	}

	if !IsNotFound(err) {
		t.Errorf("Expected IsNotFound to return true")
	}

	if IsForbidden(err) {
		t.Errorf("Expected IsForbidden to return false")
	}
}