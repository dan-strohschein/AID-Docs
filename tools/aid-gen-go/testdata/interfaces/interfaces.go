// Package interfaces demonstrates interface extraction.
package interfaces

import "io"

// Reader can read data from a source.
type Reader interface {
	// Read reads up to len(p) bytes.
	Read(p []byte) (n int, err error)
}

// Writer can write data to a destination.
type Writer interface {
	// Write writes len(p) bytes.
	Write(p []byte) (n int, err error)
}

// ReadWriter combines Reader and Writer.
type ReadWriter interface {
	Reader
	Writer
}

// Closer can release resources.
type Closer interface {
	Close() error
}

// Connection represents a network connection.
type Connection struct {
	Addr     string
	Timeout  int
	internal io.Reader
}

// Read reads data from the connection.
func (c *Connection) Read(p []byte) (int, error) {
	return 0, nil
}

// Write writes data to the connection.
func (c *Connection) Write(p []byte) (int, error) {
	return len(p), nil
}

// Close closes the connection.
func (c *Connection) Close() error {
	return nil
}

// Dial creates a new connection.
func Dial(addr string) (*Connection, error) {
	return &Connection{Addr: addr}, nil
}
