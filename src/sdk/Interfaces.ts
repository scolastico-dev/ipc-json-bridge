/**
 * Defines the set of actions that can occur during IPC communication.
 * These actions represent the different states and events that can be
 * transmitted between the main process and connected clients.
 */
export enum Action {
  /** Indicates a new client has established a connection */
  CONNECT = 'connect',
  /** Indicates an existing client has terminated their connection */
  DISCONNECT = 'disconnect',
}

/**
 * Defines the fundamental message structure used for all IPC communications.
 * This interface serves as the base for more specific message types and
 * contains all possible fields that might be used in any message.
 */
export interface BaseMessage {
  /** Unique identifier for the client connection */
  id?: string;
  /** Payload data encoded as a base64 string */
  msg?: string;
  /** When true, signals that the connection should be terminated after message processing */
  disconnect?: boolean;
  /** Specifies the type of action being performed */
  action?: Action;
  /** Process ID of the connected client (platform-dependent availability) */
  pid?: number;
  /** Description of any error that occurred */
  error?: string;
  /** Additional error context or stack trace information */
  details?: string;
  /** Path to the IPC socket being used for communication */
  socket?: string;
  /** Protocol version identifier for compatibility checking */
  version?: number;
}

/**
 * Represents a message received from a client through the IPC channel.
 * Contains the client's identifier and the actual message content.
 */
export interface IpcIncomingMessage extends Pick<Required<BaseMessage>, 'id' | 'msg'> {}

/**
 * Intermediate type definition for outgoing message structure.
 * Combines required message fields with optional disconnect flag.
 */
type IpcOutgoingMessageType = Pick<Required<BaseMessage>, 'id' | 'msg'> & Pick<BaseMessage, 'disconnect'>

/**
 * Represents a message to be sent to a client through the IPC channel.
 * Includes options for disconnecting the client after message delivery.
 */
export interface IpcOutgoingMessage extends IpcOutgoingMessageType {}

/**
 * Represents the initialization message sent when the IPC system is ready.
 * Contains essential connection details for clients to establish communication.
 */
export interface ReadyMessage extends Pick<Required<BaseMessage>, 'socket' | 'version'> {}

/**
 * Represents an error condition in the IPC system.
 * Provides both a high-level error message and detailed debugging information.
 */
export interface ErrorMessage extends Pick<Required<BaseMessage>, 'error' | 'details'> {}

/**
 * Represents the establishment of a new client connection.
 * Includes client identification and process information.
 */
export interface ConnectMessage extends Pick<Required<BaseMessage>, 'id' | 'action' | 'pid'> {
  action: Action.CONNECT;
}

/**
 * Represents the termination of an existing client connection.
 * Provides the identifier of the disconnecting client.
 */
export interface DisconnectMessage extends Pick<Required<BaseMessage>, 'id' | 'action'> {
  action: Action.DISCONNECT;
}

/**
 * Defines the set of event handlers available for the IPC bridge.
 * Each event corresponds to a specific message type that can be received.
 */
export interface IpcBridgeEvents {
  connect: (message: ConnectMessage) => void;
  disconnect: (message: DisconnectMessage) => void;
  message: (message: IpcIncomingMessage) => void;
  ready: (message: ReadyMessage) => void;
  error: (message: ErrorMessage) => void;
}

/**
 * Represents the configuration options available for the IPC bridge.
 * Allows customization of the IPC socket path and binary execution path.
 */
export interface IpcBridgeOptions {
  socketPath?: string;
  binaryPath?: string;
  asClient?: boolean;
}
