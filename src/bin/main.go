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
	"strings"
	"sync"
	"os/signal"
	"syscall"

	"github.com/google/uuid"
)

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

func logSocketPathAndVersion(socketPath string) {
  logJSON(Message{
    Socket:  socketPath,
    Version: 1,
  })
}

func main() {
	if len(os.Args) == 2 {
		socketPath := os.Args[1]
    logSocketPathAndVersion(socketPath)
    runServer(socketPath)
	} else if len(os.Args) == 3 {
    param := strings.ToLower(os.Args[1])
    socketPath := os.Args[2]
    if (param == "--client") {
      logSocketPathAndVersion(socketPath)
      runClient(socketPath)
    } else if (param == "--server") {
      logSocketPathAndVersion(socketPath)
      runServer(socketPath)
    } else {
      logError("Invalid argument", fmt.Errorf("Invalid argument: %s", os.Args[1]))
    }
  } else {
		dir := os.TempDir()
		socketPath := filepath.Join(dir, "ipc_socket_"+uuid.New().String())
    logSocketPathAndVersion(socketPath)
		runServer(socketPath)
	}
}

func runServer(socketPath string) {
	setupCleanup(socketPath)
	listener, err := createListener(socketPath)
	if err != nil {
		logError("Failed to create listener", err)
		os.Exit(1)
	}
	defer listener.Close()

	go acceptConnections(listener)

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

func runClient(socketPath string) {
	conn, err := connectToSocket(socketPath)
	if err != nil {
		logError("Failed to connect to socket", err)
		os.Exit(1)
	}
	defer conn.Close()

	clientID := uuid.New().String()

	pid := getPeerPID(conn)
	logJSON(Message{
		ID:     clientID,
		PID:    pid,
		Action: "connect",
	})

	go handleClientRead(clientID, conn)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		var msg Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			logError("Invalid JSON input", err)
			continue
		}
		handleClientInputMessage(&msg, conn, clientID)
	}
	if err := scanner.Err(); err != nil {
		logError("Error reading stdin", err)
	}

	pid = getPeerPID(conn)
	logJSON(Message{
		ID:     clientID,
		PID:    pid,
		Action: "disconnect",
	})
}

func createListener(socketPath string) (net.Listener, error) {
	if runtime.GOOS == "windows" {
		return createWindowsListener(socketPath)
	}
	return net.Listen("unix", socketPath)
}

func connectToSocket(socketPath string) (net.Conn, error) {
	if runtime.GOOS == "windows" {
		return connectToWindowsSocket(socketPath)
	}
	return net.Dial("unix", socketPath)
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

		pid := getPeerPID(conn)

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

func handleClientRead(clientID string, conn net.Conn) {
	defer func() {
		conn.Close()
		pid := getPeerPID(conn)
		logJSON(Message{
			ID:     clientID,
			PID:    pid,
			Action: "disconnect",
		})
	}()
	buf := make([]byte, 4096)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				logError(fmt.Sprintf("Read error from server %s", clientID), err)
			}
			return
		}
		if n > 0 {
			encoded := base64.StdEncoding.EncodeToString(buf[:n])
			logJSON(Message{
				ID:  clientID,
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

func handleClientInputMessage(msg *Message, conn net.Conn, clientID string) {
	data, err := base64.StdEncoding.DecodeString(msg.Msg)
	if err != nil {
		logError("Invalid base64 message", err)
		return
	}

	_, err = conn.Write(data)
	if err != nil {
		logError(fmt.Sprintf("Write error to server %s", clientID), err)
	}

	if msg.Disconnect {
		conn.Close()
	}
}

func logJSON(v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
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

func connectToWindowsSocket(pipeName string) (net.Conn, error) {
	fullPipeName := `\\.\pipe\` + pipeName
	return net.Dial("unix", fullPipeName)
}

func setupCleanup(socketPath string) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		<-sigChan
		cleanup(socketPath)
		os.Exit(0)
	}()
	if runtime.GOOS != "windows" {
		defer cleanup(socketPath)
	}
}

func cleanup(socketPath string) {
	if runtime.GOOS != "windows" {
		if _, err := os.Stat(socketPath); err == nil {
			if err := os.Remove(socketPath); err != nil {
				logError("Failed to remove socket file", err)
			}
		}
	}
}
