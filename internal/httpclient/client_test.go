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

func TestClient_Get(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/test" {
			t.Errorf("Expected /test path, got %s", r.URL.Path)
		}

		// Check for headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type header to be set")
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Expected Authorization header to be set")
		}

		// Return a test response
		resp := TestResponse{
			Message: "Success",
			Status:  "OK",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create a client with the test server URL
	config := DefaultConfig()
	config.BaseURL = server.URL
	config.Headers = map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer test-token",
	}

	client := NewClient(config)

	// Make a request
	var response TestResponse
	err := client.Get(context.Background(), "/test", &response)
	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}

	// Verify the response
	if response.Message != "Success" || response.Status != "OK" {
		t.Errorf("Unexpected response: %+v", response)
	}
}

func TestClient_Post(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/test" {
			t.Errorf("Expected /test path, got %s", r.URL.Path)
		}

		// Parse the request body
		var requestBody map[string]string
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Errorf("Error decoding request body: %v", err)
		}
		defer r.Body.Close()

		// Check the request body
		if requestBody["key"] != "value" {
			t.Errorf("Expected request body to have key=value, got %v", requestBody)
		}

		// Return a test response
		resp := TestResponse{
			Message: "Created",
			Status:  "OK",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create a client with the test server URL
	config := DefaultConfig()
	config.BaseURL = server.URL
	config.Headers = map[string]string{
		"Content-Type": "application/json",
	}

	client := NewClient(config)

	// Make a request
	requestBody := map[string]string{
		"key": "value",
	}
	var response TestResponse
	err := client.Post(context.Background(), "/test", requestBody, &response)
	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}

	// Verify the response
	if response.Message != "Created" || response.Status != "OK" {
		t.Errorf("Unexpected response: %+v", response)
	}
}

func TestClient_Middleware(t *testing.T) {
	// Create a test server
	var headerValue string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headerValue = r.Header.Get("X-Test-Header")

		// Return a test response
		resp := TestResponse{
			Message: "Success",
			Status:  "OK",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create a client with the test server URL
	config := DefaultConfig()
	config.BaseURL = server.URL

	client := NewClient(config)

	// Add middleware
	client.WithMiddleware(func(next Handler) Handler {
		return func(ctx context.Context, req *http.Request) (*http.Response, error) {
			req.Header.Set("X-Test-Header", "test-value")
			return next(ctx, req)
		}
	})

	// Make a request
	var response TestResponse
	err := client.Get(context.Background(), "/test", &response)
	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}

	// Verify the middleware was applied
	if headerValue != "test-value" {
		t.Errorf("Expected X-Test-Header to be set to test-value, got %s", headerValue)
	}
}

func TestClient_Retry(t *testing.T) {
	attempts := 0

	// Create a test server that fails the first 2 times
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++

		if attempts <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Return success on the 3rd attempt
		resp := TestResponse{
			Message: "Success after retry",
			Status:  "OK",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create a client with the test server URL and retry configuration
	config := DefaultConfig()
	config.BaseURL = server.URL
	config.RetryCount = 3
	config.RetryWaitTime = 10 * time.Millisecond // Short wait for tests

	client := NewClient(config)

	// Make a request
	var response TestResponse
	err := client.Get(context.Background(), "/test", &response)
	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}

	// Verify retry behavior
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}

	// Verify the response
	if response.Message != "Success after retry" || response.Status != "OK" {
		t.Errorf("Unexpected response: %+v", response)
	}
}

func TestClient_Error(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "Not Found", "code": 404}`))
	}))
	defer server.Close()

	// Create a client with the test server URL
	config := DefaultConfig()
	config.BaseURL = server.URL
	config.RetryCount = 0 // No retries

	client := NewClient(config)

	// Make a request
	var response TestResponse
	err := client.Get(context.Background(), "/test", &response)

	// Verify error handling
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}

	// Check if it's an API error
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("Expected APIError, got %T", err)
	}

	// Verify error details
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status code 404, got %d", apiErr.StatusCode)
	}

	// Test error helpers
	if !IsNotFound(err) {
		t.Errorf("Expected IsNotFound to return true")
	}

	if IsForbidden(err) {
		t.Errorf("Expected IsForbidden to return false")
	}
}
