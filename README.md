# IPC JSON Bridge

A lightweight, cross-platform CLI tool written in Go that enables seamless IPC (Inter-Process Communication) via JSON messages over Unix domain sockets and Windows named pipes. This tool serves as a simpler, more flexible alternative to node-ipc, with added features like sender PID tracking on Unix systems.

## Overview

IPC JSON Bridge CLI acts as a message relay between standard input/output and an IPC socket. It:

- Accepts JSON messages from stdin and forwards them to connected clients
- Receives messages from clients and outputs them as JSON to stdout
- Works across Unix, Darwin, and Windows
- Supports multiple CPU architectures
- Provides sender PID information on Unix systems
- Uses a simple, JSON-based protocol

## Installation

### NPM Package

Pre-built binaries are available in the NPM package for easy installation.

```bash
npm install ipc-json-bridge
```

### Manual Download

Standalone binaries are also available for direct download from the [releases](https://github.com/scolastico-dev/ipc-json-bridge/releases) page.

### Build from Source

Download the repository, and run the following commands to build the CLI tool from source.

```bash
# Download the go dependencies
go mod download

# Build the CLI tool
npm run build:<unix|darwin|windows>:<amd64|arm64|arm|386>

# Or build for all available platforms
npm run build:all
```

## Usage

### TypeScript SDK (Recommended)

```typescript
import { IpcBridge } from 'ipc-json-bridge';

// Initialize the bridge
const bridge = new IpcBridge();

// Handle events
bridge.on('ready', ({ socket }) => {
  console.log('Bridge ready on socket:', socket);
});

bridge.on('connect', ({ id, pid }) => {
  console.log('Client connected:', id, 'PID:', pid);
});

bridge.on('message', ({ id, msg }) => {
  console.log('Received from client:', id, Buffer.from(msg, 'base64').toString());
});

// Start the bridge
await bridge.start();

// Send messages
bridge.send({
  id: 'client-id',
  msg: Buffer.from('Hello!').toString('base64')
});

// Clean up
await bridge.stop();
```

### Protocol Specification

If you prefer to implement your own SDK, the bridge uses a simple JSON-based protocol for communication. The following sections provide details on the message format and protocol flow.

#### Message Format

```typescript
interface Message {
  // Core fields
  id?: string;          // Client identifier
  msg?: string;         // Base64-encoded payload
  disconnect?: boolean; // Request connection termination
  
  // Status fields
  action?: 'connect' | 'disconnect'; // Connection status
  pid?: number;                      // Process ID (Unix only)
  
  // System fields
  socket?: string;      // Socket path
  version?: string;     // Protocol version
  
  // Error handling
  error?: string;       // Error description
  details?: string;     // Error details
}
```

#### Protocol Flow

1. **Initialization**

   ```json
   {"socket": "/path/to/socket", "version": "1"}
   ```

2. **Client Connection**

   ```json
   {"id": "uuid", "action": "connect", "pid": 1234}
   ```

3. **Message Exchange**

   ```json
   // Outgoing
   {"id": "uuid", "msg": "base64-encoded-data"}
   
   // Optional disconnect
   {"id": "uuid", "msg": "base64-encoded-data", "disconnect": true}
   ```

4. **Client Disconnection**

   ```json
   {"id": "uuid", "action": "disconnect"}
   ```

#### Error Handling

```json
{"error": "Error description", "details": "Detailed error information"}
```

### Command Line Usage

```bash
# Start with default socket path
ipc-json-bridge

# Specify custom socket path
ipc-json-bridge /path/to/socket

# Windows named pipe
ipc-json-bridge \\.\pipe\my-pipe
```

## Example Communication Flow

```bash
# Bridge initialization
> {"socket":"/tmp/ipc_socket_uuid", "version": 1}

# Client connects
> {"id":"client-uuid","action":"connect","pid":1234}

# Send message
< {"id":"client-uuid","msg":"SGVsbG8gV29ybGQh"}

# Receive response
> {"id":"client-uuid","msg":"UmVzcG9uc2UK"}

# Disconnect client
< {"id":"client-uuid","msg":"R29vZGJ5ZQ==","disconnect":true}

# Client disconnected
> {"id":"client-uuid","action":"disconnect"}
```

## Platform-Specific Considerations

### Unix

- Socket files persist until deleted
- PID tracking available
- Standard file permissions apply

### Darwin

- Socket files persist until deleted
- PID tracking not available
- Standard file permissions apply

### Windows

- Named pipes follow `\\.\pipe\name` convention
- PID tracking not available
- Different permission model

## License

This project is licensed under the **MIT License**.

### About

MIT

A short and simple permissive license with conditions only requiring preservation of copyright and license notices. Licensed works, modifications, and larger works may be distributed under different terms and without source code.

### What you can do

| Permissions                                                                                                                       | Conditions                                                                                                                                                   | Limitations                                                                                                            |
|-----------------------------------------------------------------------------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------|------------------------------------------------------------------------------------------------------------------------|
| <details><summary>🟢 Commercial use</summary>The licensed material and derivatives may be used for commercial purposes.</details> | <details><summary>🔵 License and copyright notice</summary>A copy of the license and copyright notice must be included with the licensed material.</details> | <details><summary>🔴 Liability</summary>This license includes a limitation of liability.</details>                     |
| <details><summary>🟢 Distribution</summary>The licensed material may be distributed.</details>                                    |                                                                                                                                                              | <details><summary>🔴 Warranty</summary>This license explicitly states that it does NOT provide any warranty.</details> |
| <details><summary>🟢 Modification</summary>The licensed material may be modified.</details>                                       |                                                                                                                                                              |                                                                                                                        |
| <details><summary>🟢 Private use</summary>The licensed material may be used and modified in private.</details>                    |                                                                                                                                                              |                                                                                                                        |

*Information provided by https://choosealicense.com/licenses/mit/, this is not legal advice.*
