package docker_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TerminalMessage represents a WebSocket message for Docker terminal
type TerminalMessage struct {
	Type     string `json:"type"`
	Data     string `json:"data,omitempty"`
	Rows     int    `json:"rows,omitempty"`
	Cols     int    `json:"cols,omitempty"`
	Code     string `json:"code,omitempty"`
	Message  string `json:"message,omitempty"`
	ExitCode int    `json:"exit_code,omitempty"`
}

// TestContractTerminalSessionCreation tests the POST /api/v1/docker/instances/:id/containers/:cid/terminal endpoint contract
func TestContractTerminalSessionCreation(t *testing.T) {
	t.Run("should create terminal session successfully", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"shell": "/bin/sh",
			"rows":  30,
			"cols":  120,
			"env": map[string]string{
				"TERM": "xterm-256color",
			},
		}

		mockResponse := map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"session_id": "test-session-uuid",
				"ws_url":     "ws://localhost:12306/api/v1/docker/terminal/test-session-uuid",
				"expires_at": "2025-10-22T11:00:00Z",
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Contains(t, r.URL.Path, "/terminal")

			// Verify request body
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			assert.Equal(t, "/bin/sh", body["shell"])
			assert.Equal(t, float64(30), body["rows"].(float64))
			assert.Equal(t, float64(120), body["cols"].(float64))
			assert.NotNil(t, body["env"])

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(mockResponse)
		}))
		defer server.Close()

		bodyJSON, _ := json.Marshal(requestBody)
		resp, err := http.Post(
			server.URL+"/api/v1/docker/instances/test-uuid/containers/container-id/terminal",
			"application/json",
			strings.NewReader(string(bodyJSON)),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.True(t, result["success"].(bool))

		data := result["data"].(map[string]interface{})
		assert.NotEmpty(t, data["session_id"])
		assert.NotEmpty(t, data["ws_url"])
		assert.NotEmpty(t, data["expires_at"])
	})

	t.Run("should return error when container not running", func(t *testing.T) {
		mockError := map[string]interface{}{
			"success": false,
			"error": map[string]interface{}{
				"code":    "BAD_REQUEST",
				"message": "容器未在运行",
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(mockError)
		}))
		defer server.Close()

		requestBody := map[string]interface{}{
			"shell": "/bin/sh",
		}
		bodyJSON, _ := json.Marshal(requestBody)
		resp, err := http.Post(
			server.URL+"/api/v1/docker/instances/test-uuid/containers/stopped-container/terminal",
			"application/json",
			strings.NewReader(string(bodyJSON)),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("should return error when instance offline", func(t *testing.T) {
		mockError := map[string]interface{}{
			"success": false,
			"error": map[string]interface{}{
				"code":    "SERVICE_UNAVAILABLE",
				"message": "Docker实例离线",
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(mockError)
		}))
		defer server.Close()

		requestBody := map[string]interface{}{
			"shell": "/bin/sh",
		}
		bodyJSON, _ := json.Marshal(requestBody)
		resp, err := http.Post(
			server.URL+"/api/v1/docker/instances/offline-uuid/containers/container-id/terminal",
			"application/json",
			strings.NewReader(string(bodyJSON)),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	})
}

// TestContractWebSocketConnection tests the WebSocket connection and handshake
func TestContractWebSocketConnection(t *testing.T) {
	t.Run("should establish WebSocket connection successfully", func(t *testing.T) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify token query parameter
			token := r.URL.Query().Get("token")
			assert.NotEmpty(t, token, "JWT token should be provided in query parameter")

			// Verify session_id in path
			assert.Contains(t, r.URL.Path, "/terminal/")

			// Upgrade to WebSocket
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				t.Logf("WebSocket upgrade failed: %v", err)
				return
			}
			defer conn.Close()

			// Send welcome message (output type)
			welcomeMsg := TerminalMessage{
				Type: "output",
				Data: "Connected to container terminal\r\n",
			}
			conn.WriteJSON(welcomeMsg)

			// Wait for client messages
			for i := 0; i < 2; i++ {
				var msg TerminalMessage
				if err := conn.ReadJSON(&msg); err != nil {
					return
				}

				// Echo back or send appropriate response
				if msg.Type == "ping" {
					conn.WriteJSON(TerminalMessage{Type: "pong"})
				} else if msg.Type == "input" {
					// Echo the input as output
					conn.WriteJSON(TerminalMessage{
						Type: "output",
						Data: msg.Data,
					})
				}
			}
		}))
		defer server.Close()

		// Convert http:// to ws://
		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/docker/terminal/test-session-uuid?token=test-jwt-token"

		// Connect to WebSocket
		conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
		defer conn.Close()

		// Receive welcome message
		var welcomeMsg TerminalMessage
		err = conn.ReadJSON(&welcomeMsg)
		require.NoError(t, err)
		assert.Equal(t, "output", welcomeMsg.Type)
		assert.Contains(t, welcomeMsg.Data, "Connected")

		// Send ping and receive pong
		err = conn.WriteJSON(TerminalMessage{Type: "ping"})
		require.NoError(t, err)

		var pongMsg TerminalMessage
		err = conn.ReadJSON(&pongMsg)
		require.NoError(t, err)
		assert.Equal(t, "pong", pongMsg.Type)
	})

	t.Run("should reject connection with invalid token", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.URL.Query().Get("token")
			if token != "valid-token" {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error": "Invalid token"}`))
				return
			}
		}))
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/docker/terminal/test-session?token=invalid-token"

		_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err == nil {
			t.Fatal("Expected connection to fail with invalid token")
		}
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

// TestContractWebSocketInputMessage tests the input message protocol
func TestContractWebSocketInputMessage(t *testing.T) {
	t.Run("should send input message and receive output", func(t *testing.T) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close()

			var msg TerminalMessage
			if err := conn.ReadJSON(&msg); err != nil {
				return
			}

			// Verify input message structure
			assert.Equal(t, "input", msg.Type)
			assert.NotEmpty(t, msg.Data)

			// Send output response
			conn.WriteJSON(TerminalMessage{
				Type: "output",
				Data: "$ " + msg.Data + "hello\r\n",
			})
		}))
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/terminal"
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer conn.Close()

		// Send input message
		inputMsg := TerminalMessage{
			Type: "input",
			Data: "echo hello\n",
		}
		err = conn.WriteJSON(inputMsg)
		require.NoError(t, err)

		// Receive output message
		var outputMsg TerminalMessage
		err = conn.ReadJSON(&outputMsg)
		require.NoError(t, err)
		assert.Equal(t, "output", outputMsg.Type)
		assert.Contains(t, outputMsg.Data, "hello")
	})

	t.Run("should handle special characters in input", func(t *testing.T) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close()

			var msg TerminalMessage
			conn.ReadJSON(&msg)

			// Verify special characters (Ctrl+C = \x03)
			assert.Contains(t, []string{"\x03", "\t", "\n"}, msg.Data)

			conn.WriteJSON(TerminalMessage{
				Type: "output",
				Data: "^C\r\n",
			})
		}))
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/terminal"
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer conn.Close()

		// Send Ctrl+C
		err = conn.WriteJSON(TerminalMessage{
			Type: "input",
			Data: "\x03", // Ctrl+C
		})
		require.NoError(t, err)

		var outputMsg TerminalMessage
		conn.ReadJSON(&outputMsg)
		assert.Equal(t, "output", outputMsg.Type)
	})
}

// TestContractWebSocketResizeMessage tests the resize message protocol
func TestContractWebSocketResizeMessage(t *testing.T) {
	t.Run("should send resize message successfully", func(t *testing.T) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}

		receivedResize := false
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close()

			var msg TerminalMessage
			if err := conn.ReadJSON(&msg); err != nil {
				return
			}

			// Verify resize message structure
			assert.Equal(t, "resize", msg.Type)
			assert.Greater(t, msg.Rows, 0)
			assert.Greater(t, msg.Cols, 0)
			receivedResize = true
		}))
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/terminal"
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer conn.Close()

		// Send resize message
		resizeMsg := TerminalMessage{
			Type: "resize",
			Rows: 40,
			Cols: 150,
		}
		err = conn.WriteJSON(resizeMsg)
		require.NoError(t, err)

		// Give server time to process
		time.Sleep(100 * time.Millisecond)
		assert.True(t, receivedResize, "Server should have received resize message")
	})
}

// TestContractWebSocketPingPong tests the ping/pong heartbeat protocol
func TestContractWebSocketPingPong(t *testing.T) {
	t.Run("should respond to ping with pong", func(t *testing.T) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close()

			for {
				var msg TerminalMessage
				if err := conn.ReadJSON(&msg); err != nil {
					return
				}

				if msg.Type == "ping" {
					conn.WriteJSON(TerminalMessage{Type: "pong"})
				}
			}
		}))
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/terminal"
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer conn.Close()

		// Send multiple pings
		for i := 0; i < 3; i++ {
			err = conn.WriteJSON(TerminalMessage{Type: "ping"})
			require.NoError(t, err)

			var pongMsg TerminalMessage
			err = conn.ReadJSON(&pongMsg)
			require.NoError(t, err)
			assert.Equal(t, "pong", pongMsg.Type)
		}
	})
}

// TestContractWebSocketErrorMessage tests the error message protocol
func TestContractWebSocketErrorMessage(t *testing.T) {
	t.Run("should receive error message when exec fails", func(t *testing.T) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close()

			// Send error message
			errorMsg := TerminalMessage{
				Type:    "error",
				Code:    "EXEC_FAILED",
				Message: "Failed to execute command: container is not running",
			}
			conn.WriteJSON(errorMsg)

			// Close connection after error
			conn.Close()
		}))
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/terminal"
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer conn.Close()

		// Receive error message
		var errorMsg TerminalMessage
		err = conn.ReadJSON(&errorMsg)
		require.NoError(t, err)
		assert.Equal(t, "error", errorMsg.Type)
		assert.NotEmpty(t, errorMsg.Code)
		assert.NotEmpty(t, errorMsg.Message)
		assert.Contains(t, []string{
			"EXEC_FAILED",
			"CONTAINER_STOPPED",
			"AGENT_DISCONNECTED",
			"SESSION_TIMEOUT",
			"INTERNAL_ERROR",
		}, errorMsg.Code)
	})
}

// TestContractWebSocketExitMessage tests the exit message protocol
func TestContractWebSocketExitMessage(t *testing.T) {
	t.Run("should receive exit message when session ends", func(t *testing.T) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close()

			// Wait for input
			var msg TerminalMessage
			conn.ReadJSON(&msg)

			// If input is "exit", send exit message
			if msg.Type == "input" && strings.Contains(msg.Data, "exit") {
				exitMsg := TerminalMessage{
					Type:     "exit",
					ExitCode: 0,
				}
				conn.WriteJSON(exitMsg)
			}
		}))
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/terminal"
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer conn.Close()

		// Send exit command
		err = conn.WriteJSON(TerminalMessage{
			Type: "input",
			Data: "exit\n",
		})
		require.NoError(t, err)

		// Receive exit message
		var exitMsg TerminalMessage
		err = conn.ReadJSON(&exitMsg)
		require.NoError(t, err)
		assert.Equal(t, "exit", exitMsg.Type)
		assert.Equal(t, 0, exitMsg.ExitCode)
	})

	t.Run("should handle non-zero exit code", func(t *testing.T) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close()

			// Simulate process crash
			exitMsg := TerminalMessage{
				Type:     "exit",
				ExitCode: 137, // SIGKILL
			}
			conn.WriteJSON(exitMsg)
		}))
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/terminal"
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer conn.Close()

		// Receive exit message with non-zero code
		var exitMsg TerminalMessage
		err = conn.ReadJSON(&exitMsg)
		require.NoError(t, err)
		assert.Equal(t, "exit", exitMsg.Type)
		assert.NotEqual(t, 0, exitMsg.ExitCode)
		assert.Equal(t, 137, exitMsg.ExitCode)
	})
}

// TestContractWebSocketMessageSizeLimit tests the message size limit
func TestContractWebSocketMessageSizeLimit(t *testing.T) {
	t.Run("should handle messages within size limit", func(t *testing.T) {
		upgrader := websocket.Upgrader{
			CheckOrigin:     func(r *http.Request) bool { return true },
			ReadBufferSize:  8192,
			WriteBufferSize: 8192,
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close()

			var msg TerminalMessage
			if err := conn.ReadJSON(&msg); err != nil {
				// Message too large, should cause error
				return
			}

			// Echo back
			conn.WriteJSON(TerminalMessage{
				Type: "output",
				Data: "Received",
			})
		}))
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/terminal"
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer conn.Close()

		// Send message within limit (< 8KB)
		largeData := strings.Repeat("a", 4096) // 4KB
		err = conn.WriteJSON(TerminalMessage{
			Type: "input",
			Data: largeData,
		})
		require.NoError(t, err)

		var outputMsg TerminalMessage
		err = conn.ReadJSON(&outputMsg)
		require.NoError(t, err)
		assert.Equal(t, "output", outputMsg.Type)
	})
}

// TestContractWebSocketCompleteFlow tests a complete terminal session flow
func TestContractWebSocketCompleteFlow(t *testing.T) {
	t.Run("should handle complete terminal session lifecycle", func(t *testing.T) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close()

			// Welcome message
			conn.WriteJSON(TerminalMessage{
				Type: "output",
				Data: "Welcome to container terminal\r\n$ ",
			})

			messageCount := 0
			for {
				var msg TerminalMessage
				if err := conn.ReadJSON(&msg); err != nil {
					return
				}
				messageCount++

				switch msg.Type {
				case "input":
					if strings.Contains(msg.Data, "exit") {
						// Exit command received
						conn.WriteJSON(TerminalMessage{
							Type:     "exit",
							ExitCode: 0,
						})
						return
					}
					// Echo command
					conn.WriteJSON(TerminalMessage{
						Type: "output",
						Data: msg.Data + "command output\r\n$ ",
					})
				case "resize":
					// Acknowledge resize (no response needed in protocol)
					continue
				case "ping":
					conn.WriteJSON(TerminalMessage{Type: "pong"})
				}

				// Limit message count for test
				if messageCount > 10 {
					return
				}
			}
		}))
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/terminal"
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer conn.Close()

		// 1. Receive welcome message
		var welcomeMsg TerminalMessage
		err = conn.ReadJSON(&welcomeMsg)
		require.NoError(t, err)
		assert.Equal(t, "output", welcomeMsg.Type)
		assert.Contains(t, welcomeMsg.Data, "Welcome")

		// 2. Send resize
		err = conn.WriteJSON(TerminalMessage{
			Type: "resize",
			Rows: 30,
			Cols: 120,
		})
		require.NoError(t, err)

		// 3. Send command
		err = conn.WriteJSON(TerminalMessage{
			Type: "input",
			Data: "ls -la\n",
		})
		require.NoError(t, err)

		var outputMsg TerminalMessage
		err = conn.ReadJSON(&outputMsg)
		require.NoError(t, err)
		assert.Equal(t, "output", outputMsg.Type)

		// 4. Send ping
		err = conn.WriteJSON(TerminalMessage{Type: "ping"})
		require.NoError(t, err)

		var pongMsg TerminalMessage
		err = conn.ReadJSON(&pongMsg)
		require.NoError(t, err)
		assert.Equal(t, "pong", pongMsg.Type)

		// 5. Send exit command
		err = conn.WriteJSON(TerminalMessage{
			Type: "input",
			Data: "exit\n",
		})
		require.NoError(t, err)

		// 6. Receive exit message
		var exitMsg TerminalMessage
		err = conn.ReadJSON(&exitMsg)
		require.NoError(t, err)
		assert.Equal(t, "exit", exitMsg.Type)
		assert.Equal(t, 0, exitMsg.ExitCode)
	})
}
