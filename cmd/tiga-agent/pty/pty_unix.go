//go:build !windows

package pty

import (
	"errors"
	"os"
	"os/exec"
	"syscall"

	opty "github.com/creack/pty"
)

var _ IPty = (*Pty)(nil)

// defaultShells lists shells in order of preference
var defaultShells = []string{"zsh", "fish", "bash", "sh"}

// Pty represents a Unix pseudo-terminal
type Pty struct {
	tty *os.File
	cmd *exec.Cmd
}

// Start creates and starts a new pseudo-terminal with a shell
func Start() (IPty, error) {
	// Find available shell
	var shellPath string
	for _, sh := range defaultShells {
		shellPath, _ = exec.LookPath(sh)
		if shellPath != "" {
			break
		}
	}
	if shellPath == "" {
		return nil, errors.New("no available shell found")
	}

	// Create command with shell
	cmd := exec.Command(shellPath) // #nosec
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	// Start PTY
	tty, err := opty.Start(cmd)
	if err != nil {
		return nil, err
	}

	return &Pty{tty: tty, cmd: cmd}, nil
}

// Write writes data to the terminal
func (p *Pty) Write(data []byte) (n int, err error) {
	return p.tty.Write(data)
}

// Read reads data from the terminal
func (p *Pty) Read(data []byte) (n int, err error) {
	return p.tty.Read(data)
}

// Setsize sets the terminal window size
func (p *Pty) Setsize(cols, rows uint32) error {
	return opty.Setsize(p.tty, &opty.Winsize{
		Cols: uint16(cols),
		Rows: uint16(rows),
	})
}

// Close closes the terminal and kills the shell process
func (p *Pty) Close() error {
	// Close the TTY file descriptor first
	if err := p.tty.Close(); err != nil {
		return err
	}

	// Kill the entire process group
	return p.killChildProcess(p.cmd)
}

// killChildProcess kills the process and all its children
func (p *Pty) killChildProcess(c *exec.Cmd) error {
	if c.Process == nil {
		return nil
	}

	pgid, err := syscall.Getpgid(c.Process.Pid)
	if err != nil {
		// Fallback: kill the main process only
		return c.Process.Kill()
	}

	// Kill the whole process group
	if err := syscall.Kill(-pgid, syscall.SIGKILL); err != nil {
		return err
	}

	return c.Wait()
}
