import { spawnSync } from 'child_process';
import os from 'os';
import path from 'path';

const platform = os.platform();
const arch = os.arch();

const platformFolder = {
  darwin: 'darwin',
  linux: 'linux',
  win32: 'windows',
}[platform] || 'unsupported';

const archFolder = {
  x64: 'amd64',
  arm: 'arm',
  arm64: 'arm64',
  ia32: '386',
}[arch] || 'unsupported';

if (platform === 'darwin' && (arch === 'arm' || arch === 'ia32')) {
  console.error(`Unsupported architecture on Darwin: ${arch}`);
  process.exit(1);
}

if (platformFolder === 'unsupported' || archFolder === 'unsupported') {
  console.error(`Unsupported platform or architecture: ${platform} ${arch}`);
  process.exit(1);
}

const scriptPath = path.dirname(new URL(import.meta.url).pathname);
const executablePath = path.resolve(path.join(
  scriptPath,
  '..',
  '..',
  'bin',
  platformFolder,
  archFolder,
  platform === 'win32' ? 'ipc-json-bridge.exe' : 'ipc-json-bridge'
));

const result = spawnSync(executablePath, process.argv.slice(2), { stdio: 'inherit' });

if (result.error) {
  console.error(`Failed to execute ${executablePath}:`, result.error.message);
  process.exit(result.status || 1);
}

process.exit(result.status);
