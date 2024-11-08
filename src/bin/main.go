package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/google/uuid"
)

// Message represents the JSON message structure
type Message struct {
	ID         string `json:"id,omitempty"`
	Msg        string `json:"msg,omitempty"`
	Disconnect bool   `json:"disconnect,omitempty"`
	Action     string `json:"action,omitempty"`
	PID        int    `json:"pid,omitempty"`
	Error      string `json:"error,omitempty"`
	Details    string `json:"details,omitempty"`
	Socket     string `json:"socket,omitempty"`
	Version    int    `json:"version,omitempty"`
}

type Client struct {
	ID   string
	Conn net.Conn
}

var (
	clients   = make(map[string]*Client)
	clientsMu sync.Mutex
)

func main() {
	socketPath := ""
	if len(os.Args) > 1 {
		socketPath = os.Args[1]
	} else {
		// Generate temporary file
		dir := os.TempDir()
		socketPath = filepath.Join(dir, "ipc_socket_"+uuid.New().String())
	}

	logJSON(Message{
		Socket: socketPath,
		Version: 1,
	})

	listener, err := createListener(socketPath)
	if err != nil {
		logError("Failed to create listener", err)
		os.Exit(1)
	}
	defer listener.Close()

	// Start goroutine to accept connections
	go acceptConnections(listener)

	// Read JSON commands from stdin
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		var msg Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			logError("Invalid JSON input", err)
			continue
		}
		handleInputMessage(&msg)
	}
	if err := scanner.Err(); err != nil {
		logError("Error reading stdin", err)
	}
}

func createListener(socketPath string) (net.Listener, error) {
	if runtime.GOOS == "windows" {
		return createWindowsListener(socketPath)
	}
	return net.Listen("unix", socketPath)
}

func acceptConnections(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			logError("Accept error", err)
			return
		}
		clientID := uuid.New().String()
		client := &Client{
			ID:   clientID,
			Conn: conn,
		}
		clientsMu.Lock()
		clients[clientID] = client
		clientsMu.Unlock()

		// Get PID of the connecting process
		pid := getPeerPID(conn)

		// Log connect event
		logJSON(Message{
			ID:     clientID,
			PID:    pid,
			Action: "connect",
		})

		go handleClient(client)
	}
}

func handleClient(client *Client) {
	defer func() {
		client.Conn.Close()
		clientsMu.Lock()
		delete(clients, client.ID)
		clientsMu.Unlock()
		pid := getPeerPID(client.Conn)
		// Log disconnect event
		logJSON(Message{
			ID:     client.ID,
			PID:    pid,
			Action: "disconnect",
		})
	}()

	buf := make([]byte, 4096)
	for {
		n, err := client.Conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				logError(fmt.Sprintf("Read error from client %s", client.ID), err)
			}
			return
		}
		if n > 0 {
			encoded := base64.StdEncoding.EncodeToString(buf[:n])
			logJSON(Message{
				ID:  client.ID,
				Msg: encoded,
			})
		}
	}
}

func handleInputMessage(msg *Message) {
	clientsMu.Lock()
	client, exists := clients[msg.ID]
	clientsMu.Unlock()
	if !exists {
		logJSON(Message{
			Error:   "Client not found",
			Details: fmt.Sprintf("Client ID %s not found", msg.ID),
		})
		return
	}

	data, err := base64.StdEncoding.DecodeString(msg.Msg)
	if err != nil {
		logError("Invalid base64 message", err)
		return
	}

	_, err = client.Conn.Write(data)
	if err != nil {
		logError(fmt.Sprintf("Write error to client %s", msg.ID), err)
	}

	if msg.Disconnect {
		client.Conn.Close()
	}
}

func logJSON(v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
		// In case of JSON marshal error, print a simple JSON error message
		fmt.Printf(`{"error":"JSON marshal error","details":"%s"}`+"\n", err.Error())
		return
	}
	fmt.Println(string(b))
}

func logError(message string, err error) {
	logJSON(Message{
		Error:   message,
		Details: err.Error(),
	})
}

func createWindowsListener(pipeName string) (net.Listener, error) {
	fullPipeName := `\\.\pipe\` + pipeName
	return net.Listen("unix", fullPipeName)
}
