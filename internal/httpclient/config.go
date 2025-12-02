package httpclient

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// AuthorizationConfig represents authorization settings in the YAML configuration
type AuthorizationConfig struct {
	Type        string `yaml:"type"`          // e.g., "Bearer"
	TokenEnvVar string `yaml:"token_env_var"` // Name of environment variable for token
}

// ClientConfig represents the YAML configuration for an HTTP client
type ClientConfig struct {
	BaseURL          string               `yaml:"base_url"`
	Timeout          string               `yaml:"timeout"`
	Headers          map[string]string    `yaml:"headers"`
	Authorization    *AuthorizationConfig `yaml:"authorization,omitempty"`
	RetryCount       int                  `yaml:"retry_count"`
	RetryWaitTime    string               `yaml:"retry_wait_time"`
	MaxRetryWaitTime string               `yaml:"max_retry_wait_time"`
	EnableLogging    bool                 `yaml:"enable_logging"`
}

// APIConfigs represents a map of named API configurations
type APIConfigs struct {
	Clients map[string]ClientConfig `yaml:"clients"`
}

// LoadConfig loads client configuration from a YAML file
func LoadConfig(path string) (*APIConfigs, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var configs APIConfigs
	if err := yaml.Unmarshal(data, &configs); err != nil {
		return nil, fmt.Errorf("error parsing YAML config: %w", err)
	}

	return &configs, nil
}

// GetClientConfig returns a client config by name, with environment variable substitution
func (c *APIConfigs) GetClientConfig(name string) (*ClientConfig, error) {
	config, ok := c.Clients[name]
	if !ok {
		return nil, fmt.Errorf("client config not found: %s", name)
	}

	// Handle authorization configuration
	if config.Authorization != nil {
		// Get token from environment variable
		tokenEnvVar := config.Authorization.TokenEnvVar
		if tokenEnvVar == "" {
			return nil, fmt.Errorf("token_env_var is required in authorization configuration")
		}

		token := os.Getenv(tokenEnvVar)
		if token == "" {
			return nil, fmt.Errorf("environment variable %s for authorization token is required but not set", tokenEnvVar)
		}

		// Create authorization header with proper type
		authType := config.Authorization.Type
		if authType == "" {
			authType = "Bearer" // Default to Bearer if not specified
		}

		if config.Headers == nil {
			config.Headers = make(map[string]string)
		}

		// Set the Authorization header
		config.Headers["Authorization"] = authType + " " + token
	}

	// Replace environment variables in other header values
	for key, value := range config.Headers {
		// Skip Authorization header as it's already processed
		if key == "Authorization" && config.Authorization != nil {
			continue
		}

		// Look for ${VAR_NAME} pattern in values
		if len(value) > 3 && value[0:2] == "${" && value[len(value)-1:] == "}" {
			envName := value[2 : len(value)-1]
			envValue := os.Getenv(envName)
			if envValue == "" {
				return nil, fmt.Errorf("environment variable %s is required but not set", envName)
			}
			config.Headers[key] = envValue
		}
	}

	return &config, nil
}

// ToConfig converts a ClientConfig to a httpclient.Config
func (c *ClientConfig) ToConfig() (*Config, error) {
	config := DefaultConfig()

	// Validate required fields
	if c.BaseURL == "" {
		return nil, fmt.Errorf("base_url is required in client configuration")
	}

	config.BaseURL = c.BaseURL
	config.Headers = c.Headers
	config.RetryCount = c.RetryCount
	config.EnableLogging = c.EnableLogging

	// Parse timeout durations
	if c.Timeout == "" {
		return nil, fmt.Errorf("timeout is required in client configuration")
	}

	timeout, err := time.ParseDuration(c.Timeout)
	if err != nil {
		return nil, fmt.Errorf("invalid timeout duration: %w", err)
	}
	config.Timeout = timeout

	if c.RetryWaitTime != "" {
		retryWait, err := time.ParseDuration(c.RetryWaitTime)
		if err != nil {
			return nil, fmt.Errorf("invalid retry wait time: %w", err)
		}
		config.RetryWaitTime = retryWait
	}

	if c.MaxRetryWaitTime != "" {
		maxRetryWait, err := time.ParseDuration(c.MaxRetryWaitTime)
		if err != nil {
			return nil, fmt.Errorf("invalid max retry wait time: %w", err)
		}
		config.MaxRetryWaitTime = maxRetryWait
	}

	return config, nil
}

// CreateClient creates a new HTTP client with this configuration
func (c *ClientConfig) CreateClient() (*Client, error) {
	config, err := c.ToConfig()
	if err != nil {
		return nil, err
	}

	client := NewClient(config)

	// Add logging middleware if enabled
	if c.EnableLogging {
		client.WithMiddleware(LoggingMiddleware(false))
	}

	// Add authorization if present in headers
	authHeader, hasAuth := c.Headers["Authorization"]
	if hasAuth {
		// Remove from default headers to avoid duplication
		delete(config.Headers, "Authorization")

		// Add as middleware instead
		client.WithMiddleware(HeaderMiddleware(map[string]string{
			"Authorization": authHeader,
		}))
	}

	return client, nil
}
