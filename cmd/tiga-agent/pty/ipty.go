package pty

// IPty represents a pseudo-terminal interface
type IPty interface {
	// Write writes data to the terminal
	Write(p []byte) (n int, err error)

	// Read reads data from the terminal
	Read(p []byte) (n int, err error)

	// Setsize sets the terminal window size
	Setsize(cols, rows uint32) error

	// Close closes the terminal and kills the shell process
	Close() error
}
