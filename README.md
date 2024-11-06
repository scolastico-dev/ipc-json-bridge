# ipc-json-bridge

The ipc-json-bridge is a simple cli tool **written in Go** that allows a user,
script or program to send and receive messages to and from a
IPC socket / Windows named pipe, works on unix, darwin and windows,
as well as near any architecture.

Its intended to be an alternative approach to `node-ipc`,
being easier to use, more flexible,
and providing on unix the PID of the sender.

This is only the CLI tool, the typescript client will soon be available.

## Installation

```sh
npm i -g ipc-json-bridge
```

## Usage

```sh
ipc-json-bridge <optionally: path to socket file / named pipe>
```

## Commands

Every message is a JSON object which will follow this structure:

```go
type Message struct {
  ID         string `json:"id,omitempty"`
  Msg        string `json:"msg,omitempty"`
  Disconnect bool   `json:"disconnect,omitempty"`
  Action     string `json:"action,omitempty"`
  PID        int    `json:"pid,omitempty"`
  Error      string `json:"error,omitempty"`
  Details    string `json:"details,omitempty"`
  Socket     string `json:"socket,omitempty"`
}
```

You can send messages by following this type:

```typescript
interface Message {
  /** The id of the connection */
  id: string;
  /** The message to send in base64 */
  msg: string;
  /** If the connection should be closed after sending the message */
  disconnect?: boolean;
}
```

This can look then look like this:

```sh
StdOut: ┌─[/path/to/project]
StdOut: └─▪ node src/index.js
StdOut: {"socket":"/tmp/ipc_socket_9b5ab120-ccd0-420e-929a-cf6a1e6fc26d"}
StdOut: {"id":"5b8e49c1-dc06-4490-99be-f3612145a576","action":"connect","pid":123}
StdIn:  {"id":"5b8e49c1-dc06-4490-99be-f3612145a576","msg":"dGhpcyBpcyBhbiBleGFtcGxlIQ=="}
StdOut: {"id":"b8e49c1-dc06-4490-99be-f3612145a576","msg":"SGVsbG8gZnJvbSB0aGUgb3RoZXIgc2lkZSEK"}
StdIn:  {"id":"5b8e49c1-dc06-4490-99be-f3612145a576","msg":"QnllIQ==", "disconnect": true}
StdOut: {"id":"5b8e49c1-dc06-4490-99be-f3612145a576","action":"disconnect"}
```
