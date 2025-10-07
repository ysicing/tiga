package terminal

import (
	"fmt"
	"io"

	"golang.org/x/crypto/ssh"
)

// SSHExecutor handles SSH connections and command execution
type SSHExecutor struct {
	config *ssh.ClientConfig
}

// NewSSHExecutor creates a new SSH executor
func NewSSHExecutor(user, password string) *SSHExecutor {
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Accept any host key
	}

	return &SSHExecutor{
		config: config,
	}
}

// NewSSHExecutorWithKey creates SSH executor with private key authentication
func NewSSHExecutorWithKey(user string, privateKey []byte) (*SSHExecutor, error) {
	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	return &SSHExecutor{
		config: config,
	}, nil
}

// Connect establishes SSH connection to the host
func (e *SSHExecutor) Connect(host string, port int) (*ssh.Client, error) {
	address := fmt.Sprintf("%s:%d", host, port)
	client, err := ssh.Dial("tcp", address, e.config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH server: %w", err)
	}
	return client, nil
}

// ExecuteCommand executes a single command on the remote host
func (e *SSHExecutor) ExecuteCommand(client *ssh.Client, command string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		return string(output), fmt.Errorf("command execution failed: %w", err)
	}

	return string(output), nil
}

// StartInteractiveShell starts an interactive shell session
func (e *SSHExecutor) StartInteractiveShell(client *ssh.Client, stdin io.Reader, stdout, stderr io.Writer, width, height int) error {
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Set terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // Enable echoing
		ssh.TTY_OP_ISPEED: 14400, // Input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // Output speed = 14.4kbaud
	}

	// Request pseudo terminal
	if err := session.RequestPty("xterm-256color", height, width, modes); err != nil {
		return fmt.Errorf("request for pseudo terminal failed: %w", err)
	}

	// Connect I/O
	session.Stdin = stdin
	session.Stdout = stdout
	session.Stderr = stderr

	// Start shell
	if err := session.Shell(); err != nil {
		return fmt.Errorf("failed to start shell: %w", err)
	}

	// Wait for session to complete
	if err := session.Wait(); err != nil {
		return fmt.Errorf("shell session failed: %w", err)
	}

	return nil
}

// ResizeTerminal resizes the terminal window
func (e *SSHExecutor) ResizeTerminal(session *ssh.Session, width, height int) error {
	return session.WindowChange(height, width)
}

// Session represents an active SSH session
type Session struct {
	client  *ssh.Client
	session *ssh.Session
}

// NewSession creates a new SSH session with terminal support
func (e *SSHExecutor) NewSession(host string, port int, width, height int) (*Session, error) {
	client, err := e.Connect(host, port)
	if err != nil {
		return nil, err
	}

	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	// Request PTY
	if err := session.RequestPty("xterm-256color", height, width, modes); err != nil {
		session.Close()
		client.Close()
		return nil, fmt.Errorf("request for pseudo terminal failed: %w", err)
	}

	return &Session{
		client:  client,
		session: session,
	}, nil
}

// StartShell starts the shell for this session
func (s *Session) StartShell(stdin io.Reader, stdout, stderr io.Writer) error {
	s.session.Stdin = stdin
	s.session.Stdout = stdout
	s.session.Stderr = stderr

	if err := s.session.Shell(); err != nil {
		return fmt.Errorf("failed to start shell: %w", err)
	}

	return nil
}

// Wait waits for the session to complete
func (s *Session) Wait() error {
	return s.session.Wait()
}

// Resize resizes the terminal
func (s *Session) Resize(width, height int) error {
	return s.session.WindowChange(height, width)
}

// Close closes the session and connection
func (s *Session) Close() error {
	if s.session != nil {
		s.session.Close()
	}
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}
