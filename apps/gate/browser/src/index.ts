import { createServer } from 'node:http';

import { sidecar } from './server.ts';
import { Sessions } from './sessions.ts';

const HOST = '127.0.0.1';
const PORT = Number.parseInt(process.env.GATE_BROWSER_PORT ?? '9001', 10);
const REAPER_INTERVAL_MS = 30 * 1000;

const sessions = new Sessions(process.env.CREDENTIALS_DIR ?? '/credentials');

// invariant: bound to 127.0.0.1 inside the gate container, so the trust boundary is the container: anything in it may drive any session, nothing on the host or in a world can reach it
const server = createServer(sidecar(sessions));

const reaper = setInterval(() => {
    void sessions.reap(Date.now());
}, REAPER_INTERVAL_MS);

async function shutdown(): Promise<void> {
    clearInterval(reaper);
    server.close();
    await sessions.shutdown();
}

process.once('SIGTERM', () => {
    void shutdown();
});
process.once('SIGINT', () => {
    void shutdown();
});

server.listen(PORT, HOST, () => {
    process.stderr.write(`gate-browser listening on ${HOST}:${PORT}\n`);
});
