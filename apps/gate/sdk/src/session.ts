import { fieldOf, parsedOrText } from './json.ts';
import type { Json } from './json.ts';

const SIDECAR = 'http://127.0.0.1:9001';

type Options = Record<string, unknown>;

async function call(method: 'DELETE' | 'POST', url: string, body?: Json): Promise<Json> {
    const response = await fetch(url, {
        headers: { 'content-type': 'application/json' },
        method,
        ...(body === undefined ? {} : { body: JSON.stringify(body) }),
    });
    const text = await response.text();
    if (response.status >= 400) {
        const reason = fieldOf(parsedOrText(text), 'error');
        throw new Error(
            `${method} ${url} → ${response.status}: ${typeof reason === 'string' ? reason : text}`,
        );
    }
    return text === '' ? null : parsedOrText(text);
}

/** A browser session of the gate-browser sidecar: a cookie-loaded Chromium page a tool drives over HTTP. */
export class Session {
    readonly id: string;
    private readonly base: string;
    private ended = false;

    constructor(id: string, base: string) {
        this.id = id;
        this.base = base;
    }

    async goto(url: string, options: Options = {}): Promise<Json> {
        return await this.post('goto', { url, ...options });
    }

    async waitResponse(urlPattern: string, options: Options = {}): Promise<Json> {
        return await this.post('wait-response', { url_pattern: urlPattern, ...options });
    }

    async click(selector: string, options: Options = {}): Promise<Json> {
        return await this.post('click', { selector, ...options });
    }

    async type(selector: string, text: string, options: Options = {}): Promise<Json> {
        return await this.post('type', { selector, text, ...options });
    }

    async scroll(options: Options = {}): Promise<Json> {
        return await this.post('scroll', options);
    }

    async waitSelector(selector: string, options: Options = {}): Promise<Json> {
        return await this.post('wait-selector', { selector, ...options });
    }

    async eval(script: string): Promise<Json> {
        return await this.post('eval', { script });
    }

    async capturedResponses(options: Options = {}): Promise<Json> {
        return await this.post('captured-responses', options);
    }

    /** Close the session; a second call is a no-op. */
    async end(): Promise<void> {
        if (this.ended) {
            return;
        }
        this.ended = true;
        await call('DELETE', `${this.base}/sessions/${this.id}`);
    }

    private async post(action: string, body: Json): Promise<Json> {
        return await call('POST', `${this.base}/sessions/${this.id}/${action}`, body);
    }
}

/** Open a session on the sidecar, its browser context loaded with the cookies of `provider`. */
export async function openSession(provider: string): Promise<Session> {
    if (provider === '') {
        throw new Error('openSession: provider name required');
    }
    const base = process.env.GATE_BROWSER_URL ?? SIDECAR;
    const opened = await call('POST', `${base}/sessions`, { provider });
    return new Session(String(fieldOf(opened, 'id')), base);
}
