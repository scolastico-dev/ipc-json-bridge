import { spawn, ChildProcess } from 'child_process';
import { join, resolve } from 'path';
import { existsSync, statSync, chmodSync } from 'fs';
import { platform, arch } from 'os';
import { EventEmitter } from 'events';
import {
  Action,
  BaseMessage,
  IpcIncomingMessage,
  IpcOutgoingMessage,
  ReadyMessage,
  ErrorMessage,
  ConnectMessage,
  DisconnectMessage,
  IpcBridgeEvents,
  IpcBridgeOptions,
} from './Interfaces';

/**
 * Represents a bi-directional IPC bridge for inter-process communication.
 * Provides methods for starting and stopping the bridge, as well as sending messages.
 */
export class IpcBridge extends EventEmitter {
  private process: ChildProcess | null = null;
  private ready = false;
  private readonly binaryPath: string;
  private socketPath: string;
  private isClient: boolean;
  private stopping = false;
  private startTimeout: NodeJS.Timeout | null = null;

  constructor(options: IpcBridgeOptions = {}) {
    super();
    this.isClient = options.asClient || false;
    this.socketPath = options.socketPath || '';
    if (this.isClient && !this.socketPath) throw new Error('Client mode requires a socket path');
    this.binaryPath = this.resolveBinaryPath(options.binaryPath);
    if (!existsSync(this.binaryPath)) {
      throw new Error(`IPC bridge binary not found: ${this.binaryPath}`);
    }
  }

  // Type-safe event emitter methods
  public on<K extends keyof IpcBridgeEvents>(
    event: K,
    listener: IpcBridgeEvents[K]
  ): this {
    return super.on(event, listener);
  }

  public once<K extends keyof IpcBridgeEvents>(
    event: K,
    listener: IpcBridgeEvents[K]
  ): this {
    return super.once(event, listener);
  }

  public emit<K extends keyof IpcBridgeEvents>(
    event: K,
    ...args: Parameters<IpcBridgeEvents[K]>
  ): boolean {
    return super.emit(event, ...args);
  }

  public off<K extends keyof IpcBridgeEvents>(
    event: K,
    listener: IpcBridgeEvents[K]
  ): this {
    return super.off(event, listener);
  }

  public removeListener<K extends keyof IpcBridgeEvents>(
    event: K,
    listener: IpcBridgeEvents[K]
  ): this {
    return super.removeListener(event, listener);
  }

  public addListener<K extends keyof IpcBridgeEvents>(
    event: K,
    listener: IpcBridgeEvents[K]
  ): this {
    return super.addListener(event, listener);
  }

  private resolveBinaryPath(customPath?: string): string {
    if (customPath) return customPath;

    const platformMap: Record<string, string> = {
      darwin: 'darwin',
      linux: 'linux',
      win32: 'windows'
    };

    const archMap: Record<string, string> = {
      x64: 'amd64',
      arm: 'arm',
      arm64: 'arm64',
      ia32: '386'
    };

    const platformFolder = platformMap[platform()] || 'unsupported';
    const archFolder = archMap[arch()] || 'unsupported';

    if (platformFolder === 'unsupported' || archFolder === 'unsupported') {
      throw new Error(`Unsupported platform or architecture: ${platform()} ${arch()}`);
    }

    const executableName = platform() === 'win32' ? 'ipc-json-bridge.exe' : 'ipc-json-bridge';
    return resolve(join(__dirname, '..', '..', 'bin', platformFolder, archFolder, executableName));
  }

  public async start(): Promise<void> {
    if (this.process) {
      throw new Error('IPC bridge is already running');
    }

    const stats = statSync(this.binaryPath);
    if (!(stats.mode & 0o100)) chmodSync(this.binaryPath, stats.mode | 0o100);

    return new Promise((resolve, reject) => {
      const args = [
        this.isClient ? '--client' : '--server',
        ...(this.socketPath ? [this.socketPath] : []),
      ];
      this.stopping = false;
      this.process = spawn(this.binaryPath, args);

      const cleanup = (error?: Error) => {
        if (this.startTimeout) {
          clearTimeout(this.startTimeout);
          this.startTimeout = null;
        }
        this.process = null;
        this.ready = false;
        this.stopping = false;

        if (error) {
          reject(error);
        }
      };

      this.process.stdout?.on('data', (data: Buffer) => {
        this.handleProcessOutput(data.toString());
      });

      this.process.stderr?.on('data', (data: Buffer) => {
        if (!this.stopping) {
          this.emit('error', {
            error: 'Bridge process error',
            details: data.toString()
          });
        }
      });

      this.process.on('error', (error: Error) => {
        if (!this.stopping) {
          this.emit('error', {
            error: 'Failed to start bridge process',
            details: error.message
          });
        }
        cleanup(error);
      });

      this.process.on('exit', (code: number | null) => {
        const wasReady = this.ready;
        if (!this.stopping && wasReady && code !== 0) {
          this.emit('error', {
            error: 'Bridge process exited unexpectedly',
            details: `Exit code: ${code}`
          });
        }
        cleanup();
      });

      this.startTimeout = setTimeout(() => {
        cleanup(new Error('Bridge failed to start within timeout'));
      }, 5000);

      this.once('ready', () => {
        if (this.startTimeout) {
          clearTimeout(this.startTimeout);
          this.startTimeout = null;
        }
        resolve();
      });
    });
  }

  public async stop(): Promise<void> {
    if (!this.process) return;

    this.stopping = true;
    return new Promise<void>((resolve) => {
      const forceKillTimeout = setTimeout(() => {
        if (this.process) {
          this.process.kill('SIGKILL');
          this.process = null;
          this.ready = false;
          this.stopping = false;
          resolve();
        }
      }, 1000);

      const cleanup = () => {
        clearTimeout(forceKillTimeout);
        this.process = null;
        this.ready = false;
        this.stopping = false;
        resolve();
      };

      this.process?.once('exit', cleanup);
      this.process?.kill();
    });
  }

  private handleProcessOutput(data: string): void {
    try {
      const messages = data.trim().split('\n');
      for (const message of messages) {
        const parsed = JSON.parse(message) as BaseMessage;

        if (parsed.socket && parsed.version) {
          if (parsed.version !== 1) {
            this.emit('error', {
              error: 'Unsupported bridge version',
              details: `Expected version 1, got ${parsed.version}`
            });
            return;
          }
          this.socketPath = parsed.socket;
          this.ready = true;
          this.emit('ready', parsed as ReadyMessage);
        } else if (parsed.error) {
          this.emit('error', parsed as ErrorMessage);
        } else if (parsed.action === Action.CONNECT) {
          this.emit('connect', parsed as ConnectMessage);
        } else if (parsed.action === Action.DISCONNECT) {
          this.emit('disconnect', parsed as DisconnectMessage);
        } else if (parsed.id && parsed.msg) {
          this.emit('message', parsed as IpcIncomingMessage);
        }
      }
    } catch (error) {
      this.emit('error', {
        error: 'Failed to parse bridge output',
        details: (error as Error).message
      });
    }
  }

  public send(message: IpcOutgoingMessage): void {
    if (!this.process || !this.ready) {
      throw new Error('Bridge is not running or not ready');
    }

    const jsonMessage = JSON.stringify(message) + '\n';
    this.process.stdin?.write(jsonMessage);
  }

  public getSocketPath(): string {
    return this.socketPath;
  }

  public isReady(): boolean {
    return this.ready && this.process !== null;
  }
}
