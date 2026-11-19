import { readFile } from 'node:fs/promises';
import { createServer } from 'node:http';
import type { IncomingMessage, ServerResponse } from 'node:http';
import { join } from 'node:path';

// invariant: the GitHub API a run talks to, answered from the run's own files: a GET of
// /repos/<owner>/<repo>/contents/<folder> answers answers/github/<the same path>.json of
// the document's working directory, `{origin}` in it this server's own address (a
// release asset's download address), a path with no such file answers the file at that
// path itself as bytes (a release's asset, /repos/<owner>/<repo>/releases/assets/<id>),
// and anything without an answer file is a 404.
const root = join(process.env.TEST_WORKDIR ?? '.', 'answers', 'github');

async function read(file: string): Promise<Buffer | undefined> {
    return await readFile(file).catch(() => undefined);
}

async function answer(request: IncomingMessage, response: ServerResponse): Promise<void> {
    const path = new URL(request.url ?? '/', 'http://stub').pathname.replace(/\/+$/u, '');
    const json = await read(join(root, `${path}.json`));
    const bytes = json ?? (await read(join(root, path)));
    if (bytes === undefined) {
        response.writeHead(404);
        response.end();
        return;
    }
    response.writeHead(200, {
        'content-type': json === undefined ? 'application/octet-stream' : 'application/json',
    });
    const origin = `http://${request.headers.host ?? '127.0.0.1'}`;
    response.end(json === undefined ? bytes : json.toString('utf8').replaceAll('{origin}', origin));
}

const server = createServer((request, response) => {
    void answer(request, response);
});

server.listen(0, '127.0.0.1', () => {
    const address = server.address();
    const port = typeof address === 'object' && address !== null ? address.port : 0;
    process.stdout.write(`listening on port ${port}\n`);
});
