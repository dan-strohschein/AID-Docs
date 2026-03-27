// Package basic provides basic types and functions for testing AID extraction.
package basic

import "errors"

// MaxRetries is the maximum number of retry attempts.
const MaxRetries = 3

// DefaultTimeout is the default timeout in seconds.
const DefaultTimeout = 30.0

// ErrNotFound is returned when an item cannot be found.
var ErrNotFound = errors.New("not found")

// ErrTimeout is returned when an operation times out.
var ErrTimeout = errors.New("timeout")

// Config holds configuration for the service.
type Config struct {
	// Host is the server hostname.
	Host string
	// Port is the server port number.
	Port int
	// Debug enables debug logging.
	Debug bool
	// internal fields are not exported
	logger interface{}
}

// Validate checks if the config is valid.
func (c Config) Validate() error {
	if c.Host == "" {
		return errors.New("host is required")
	}
	return nil
}

// SetHost updates the hostname.
func (c *Config) SetHost(host string) {
	c.Host = host
}

// New creates a new Config with defaults.
func New(host string, port int) *Config {
	return &Config{Host: host, Port: port}
}

// Get retrieves a value by key from the store.
func Get(key string) (string, error) {
	return "", nil
}

// Set stores a value with the given key.
func Set(key string, value string) error {
	return nil
}

// BatchSet stores multiple key-value pairs.
func BatchSet(pairs map[string]string) (int, error) {
	return len(pairs), nil
}

// internal function - should not be exported
func internalHelper() {}
