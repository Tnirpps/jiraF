package httpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// APIError represents an error returned by an API
type APIError struct {
	StatusCode int
	Status     string
	Body       string
	Err        error
	RawBody    []byte
	Response   *http.Response
}

// NewAPIError creates a new APIError from an HTTP response
func NewAPIError(resp *http.Response) *APIError {
	// Read the response body
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()

	// Copy the body back into the response for potential re-reading
	resp.Body = io.NopCloser(bytes.NewReader(body))

	// Create the API error
	apiErr := &APIError{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		RawBody:    body,
		Response:   resp,
	}

	// Set the body as a string
	apiErr.Body = string(body)

	// Set the underlying error
	if err != nil {
		apiErr.Err = fmt.Errorf("error reading response body: %w", err)
	} else {
		apiErr.Err = fmt.Errorf("API request failed with status code: %d", resp.StatusCode)
	}

	return apiErr
}

// Error implements the error interface
func (e *APIError) Error() string {
	if e.Body != "" {
		return fmt.Sprintf("API error: %s - %s", e.Status, e.Body)
	}
	return fmt.Sprintf("API error: %s", e.Status)
}

// Unwrap returns the underlying error
func (e *APIError) Unwrap() error {
	return e.Err
}

// IsStatus checks if the error is an API error with the given status code
func IsStatus(err error, statusCode int) bool {
	apiErr, ok := err.(*APIError)
	if !ok {
		return false
	}
	return apiErr.StatusCode == statusCode
}

// IsNotFound checks if the error is a 404 Not Found error
func IsNotFound(err error) bool {
	return IsStatus(err, http.StatusNotFound)
}

// IsForbidden checks if the error is a 403 Forbidden error
func IsForbidden(err error) bool {
	return IsStatus(err, http.StatusForbidden)
}

// IsUnauthorized checks if the error is a 401 Unauthorized error
func IsUnauthorized(err error) bool {
	return IsStatus(err, http.StatusUnauthorized)
}

// IsBadRequest checks if the error is a 400 Bad Request error
func IsBadRequest(err error) bool {
	return IsStatus(err, http.StatusBadRequest)
}

// GetAPIError tries to cast an error to an APIError
func GetAPIError(err error) (*APIError, bool) {
	apiErr, ok := err.(*APIError)
	return apiErr, ok
}

// ParseErrorBody attempts to parse the error body as JSON into the given struct
func ParseErrorBody(err error, v interface{}) error {
	apiErr, ok := GetAPIError(err)
	if !ok {
		return fmt.Errorf("not an API error")
	}

	if err := json.Unmarshal(apiErr.RawBody, v); err != nil {
		return fmt.Errorf("error parsing error body: %w", err)
	}
	return nil
}
