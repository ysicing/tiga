package webssh

import (
	"encoding/json"
	"time"
)

// MessageType defines the type of WebSocket message
type MessageType string

const (
	// Client to Server messages
	MessageTypeInput   MessageType = "input"   // Terminal input
	MessageTypeResize  MessageType = "resize"  // Terminal resize
	MessageTypePing    MessageType = "ping"    // Keep-alive ping
	MessageTypeCommand MessageType = "command" // Special commands (e.g., close)

	// Server to Client messages
	MessageTypeOutput       MessageType = "output"       // Terminal output
	MessageTypePong         MessageType = "pong"         // Keep-alive pong
	MessageTypeError        MessageType = "error"        // Error message
	MessageTypeConnected    MessageType = "connected"    // Connection established
	MessageTypeDisconnected MessageType = "disconnected" // Connection closed
	MessageTypeInfo         MessageType = "info"         // Information message
)

// Message represents a WebSocket message
type Message struct {
	Type      MessageType     `json:"type"`
	Data      json.RawMessage `json:"data,omitempty"`
	Timestamp int64           `json:"timestamp"`
	SessionID string          `json:"session_id,omitempty"`
}

// InputMessage represents terminal input from client
type InputMessage struct {
	Input string `json:"input"` // Base64 encoded for binary safety
}

// OutputMessage represents terminal output to client
type OutputMessage struct {
	Output string `json:"output"` // Base64 encoded for binary safety
}

// ResizeMessage represents terminal resize event
type ResizeMessage struct {
	Cols int `json:"cols"`
	Rows int `json:"rows"`
}

// ErrorMessage represents an error message
type ErrorMessage struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// InfoMessage represents an information message
type InfoMessage struct {
	Message string `json:"message"`
	Level   string `json:"level"` // "info", "warning", "success"
}

// CommandMessage represents a command from client
type CommandMessage struct {
	Command string          `json:"command"` // "close", "pause", "resume"
	Args    json.RawMessage `json:"args,omitempty"`
}

// ConnectedMessage represents successful connection
type ConnectedMessage struct {
	SessionID string `json:"session_id"`
	HostName  string `json:"host_name"`
	HostID    string `json:"host_id"`
	Cols      int    `json:"cols"`
	Rows      int    `json:"rows"`
}

// NewMessage creates a new WebSocket message
func NewMessage(msgType MessageType, data interface{}) (*Message, error) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return &Message{
		Type:      msgType,
		Data:      dataBytes,
		Timestamp: time.Now().UnixMilli(),
	}, nil
}

// ParseMessage parses a WebSocket message
func ParseMessage(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// GetInputMessage extracts input data from message
func (m *Message) GetInputMessage() (*InputMessage, error) {
	if m.Type != MessageTypeInput {
		return nil, ErrInvalidMessageType
	}

	var input InputMessage
	if err := json.Unmarshal(m.Data, &input); err != nil {
		return nil, err
	}
	return &input, nil
}

// GetResizeMessage extracts resize data from message
func (m *Message) GetResizeMessage() (*ResizeMessage, error) {
	if m.Type != MessageTypeResize {
		return nil, ErrInvalidMessageType
	}

	var resize ResizeMessage
	if err := json.Unmarshal(m.Data, &resize); err != nil {
		return nil, err
	}
	return &resize, nil
}

// GetCommandMessage extracts command data from message
func (m *Message) GetCommandMessage() (*CommandMessage, error) {
	if m.Type != MessageTypeCommand {
		return nil, ErrInvalidMessageType
	}

	var cmd CommandMessage
	if err := json.Unmarshal(m.Data, &cmd); err != nil {
		return nil, err
	}
	return &cmd, nil
}
