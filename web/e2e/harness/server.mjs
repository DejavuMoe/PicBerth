import { spawn, spawnSync } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const repoRoot = fileURLToPath(new URL('../../../', import.meta.url));
const port = process.env.PICBERTH_PORT || '5210';
const binPath = path.join(repoRoot, 'dist', process.platform === 'win32' ? 'picberth.exe' : 'picberth');
if (!fs.existsSync(binPath)) throw new Error(`Missing ${binPath}; run 'mise run build' first.`);

// Do not let the developer's config redirect test storage, assets, or image URLs.
const env = Object.fromEntries(Object.entries(process.env).filter(([key]) => !key.startsWith('PICBERTH_')));
const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'picberth-e2e-'));
console.log(`[e2e-server] Temporary data: ${tempDir}`);
process.once('exit', () => fs.rmSync(tempDir, { recursive: true, force: true }));

const init = spawnSync(binPath,
  ['admin', 'create', '--data-dir', tempDir, '--username', 'testadmin', '--password-stdin'],
  { input: 'TestPass123!\n', stdio: ['pipe', 'inherit', 'inherit'], env });
if (init.error) throw init.error;
if (init.status !== 0) throw new Error(`Admin initialization failed: ${init.status}`);

const server = spawn(binPath,
  ['serve', '--data-dir', tempDir, '--listen', `127.0.0.1:${port}`, '--public-url', `http://127.0.0.1:${port}`, '--log-level', 'warn'],
  { stdio: 'inherit', env });

// Wait for Go to close the database and workers before the exit hook removes data.
for (const signal of ['SIGINT', 'SIGTERM']) process.on(signal, () => server.kill('SIGTERM'));
server.once('error', (error) => {
  console.error(error);
  process.exitCode = 1;
});
server.once('close', (code) => { process.exitCode = process.exitCode || code || 0; });
