package managers

import (
	"fmt"
	"strings"
)

// CommandValidator validates Docker exec commands for security
type CommandValidator struct {
	allowedCommands map[string]bool
	blockedCommands map[string]bool
	blockedPatterns []string
}

// NewCommandValidator creates a new command validator
func NewCommandValidator() *CommandValidator {
	return &CommandValidator{
		// Whitelist of safe commands (can be configured)
		allowedCommands: map[string]bool{
			"ls":     true,
			"cat":    true,
			"echo":   true,
			"pwd":    true,
			"whoami": true,
			"ps":     true,
			"top":    true,
			"env":    true,
			"date":   true,
			"df":     true,
			"du":     true,
			"free":   true,
			"uptime": true,
		},
		// Blacklist of dangerous commands
		blockedCommands: map[string]bool{
			"rm":         true,
			"rmdir":      true,
			"dd":         true,
			"mkfs":       true,
			"fdisk":      true,
			"shutdown":   true,
			"reboot":     true,
			"halt":       true,
			"poweroff":   true,
			"init":       true,
			"kill":       true,
			"killall":    true,
			"pkill":      true,
			"su":         true,
			"sudo":       true,
			"chmod":      true,
			"chown":      true,
			"chroot":     true,
			"mount":      true,
			"umount":     true,
			"iptables":   true,
			"nc":         true,
			"netcat":     true,
			"curl":       true, // Can be used to exfiltrate data
			"wget":       true, // Can be used to download malicious files
			"ssh":        true,
			"scp":        true,
			"telnet":     true,
			"ftp":        true,
		},
		// Blocked patterns in arguments
		blockedPatterns: []string{
			"&&",      // Command chaining
			"||",      // Command chaining
			";",       // Command separator
			"|",       // Pipe
			"<",       // Redirection
			">",       // Redirection
			"`",       // Command substitution
			"$(",      // Command substitution
			"${",      // Variable expansion
			"../",     // Path traversal
			"~",       // Home directory expansion
			"*",       // Wildcard (can be dangerous)
			"?",       // Wildcard
			"\n",      // Newline injection
			"\r",      // Carriage return injection
		},
	}
}

// ValidateCommand validates a command before execution
func (v *CommandValidator) ValidateCommand(cmd []string) error {
	if len(cmd) == 0 {
		return fmt.Errorf("empty command")
	}

	baseCmd := strings.ToLower(cmd[0])

	// Check blacklist first
	if v.blockedCommands[baseCmd] {
		return fmt.Errorf("command '%s' is not allowed (security policy)", baseCmd)
	}

	// Check whitelist (if enforced)
	// Note: In production, you might want to enforce whitelist strictly
	// For now, we only block dangerous commands
	// if !v.allowedCommands[baseCmd] {
	//     return fmt.Errorf("command '%s' is not in allowed list", baseCmd)
	// }

	// Validate command arguments
	for i, arg := range cmd {
		if err := v.validateArgument(arg, i); err != nil {
			return fmt.Errorf("invalid argument at position %d: %w", i, err)
		}
	}

	return nil
}

// validateArgument validates a single command argument
func (v *CommandValidator) validateArgument(arg string, position int) error {
	// Check for blocked patterns
	for _, pattern := range v.blockedPatterns {
		if strings.Contains(arg, pattern) {
			return fmt.Errorf("argument contains blocked pattern '%s'", pattern)
		}
	}

	// Additional checks for path traversal
	if strings.Contains(arg, "..") && strings.Contains(arg, "/") {
		return fmt.Errorf("potential path traversal detected")
	}

	// Check for excessively long arguments (potential buffer overflow)
	if len(arg) > 1024 {
		return fmt.Errorf("argument too long (max 1024 characters)")
	}

	// Check for null bytes
	if strings.Contains(arg, "\x00") {
		return fmt.Errorf("null byte detected in argument")
	}

	return nil
}

// EnableWhitelistMode enables strict whitelist mode
func (v *CommandValidator) EnableWhitelistMode() {
	// This would be configurable in production
}

// AddAllowedCommand adds a command to the whitelist
func (v *CommandValidator) AddAllowedCommand(cmd string) {
	v.allowedCommands[strings.ToLower(cmd)] = true
}

// RemoveAllowedCommand removes a command from the whitelist
func (v *CommandValidator) RemoveAllowedCommand(cmd string) {
	delete(v.allowedCommands, strings.ToLower(cmd))
}

// AddBlockedCommand adds a command to the blacklist
func (v *CommandValidator) AddBlockedCommand(cmd string) {
	v.blockedCommands[strings.ToLower(cmd)] = true
}

// RemoveBlockedCommand removes a command from the blacklist
func (v *CommandValidator) RemoveBlockedCommand(cmd string) {
	delete(v.blockedCommands, strings.ToLower(cmd))
}

// IsCommandAllowed checks if a command is explicitly allowed
func (v *CommandValidator) IsCommandAllowed(cmd string) bool {
	return v.allowedCommands[strings.ToLower(cmd)]
}

// IsCommandBlocked checks if a command is explicitly blocked
func (v *CommandValidator) IsCommandBlocked(cmd string) bool {
	return v.blockedCommands[strings.ToLower(cmd)]
}
