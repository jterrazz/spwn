import { randomUUID } from 'node:crypto';
import { chromium } from 'playwright';
import type { Browser } from 'playwright';

import { providerCookies } from './cookies.ts';
import { RefusedError } from './http.ts';
import { Session } from './session.ts';

const USER_AGENT =
    'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36';

// Stryker disable next-line ObjectLiteral,ArrayDeclaration,StringLiteral,BooleanLiteral: a mutant turning headless off would open a window on the machine measuring it, and the flags are what Chromium needs inside the gate container, changing nothing outside one
const LAUNCH = { args: ['--no-sandbox', '--disable-dev-shm-usage'], headless: true };

/** Every open session over one warm Chromium, which every session shares. */
export class Sessions {
    private browser: Browser | null = null;
    private readonly credentials: string;
    private readonly open = new Map<string, Session>();

    constructor(credentials: string) {
        this.credentials = credentials;
    }

    get size(): number {
        return this.open.size;
    }

    get connected(): boolean {
        return this.browser?.isConnected() === true;
    }

    async create(provider: string): Promise<{ id: string; session: Session }> {
        const cookies = providerCookies(this.credentials, provider);
        const browser = await this.chromium();
        const context = await browser.newContext({
            userAgent: USER_AGENT,
            viewport: { height: 900, width: 1280 },
        });
        await context.addCookies(cookies);
        const id = randomUUID();
        const session = new Session(context, await context.newPage());
        this.open.set(id, session);
        return { id, session };
    }

    get(id: string): Session {
        const session = this.open.get(id);
        if (session === undefined) {
            throw new RefusedError(404, `unknown session "${id}"`);
        }
        session.touch();
        return session;
    }

    async end(id: string): Promise<void> {
        await this.open.get(id)?.close();
        this.open.delete(id);
    }

    async reap(now: number): Promise<void> {
        const expired = [...this.open].filter(([, session]) => session.expired(now));
        await Promise.all(
            expired.map(async ([id]) => {
                await this.end(id);
            }),
        );
    }

    async shutdown(): Promise<void> {
        await Promise.all(
            [...this.open.keys()].map(async (id) => {
                await this.end(id);
            }),
        );
        await this.browser?.close().catch(() => undefined);
    }

    private async chromium(): Promise<Browser> {
        if (this.browser?.isConnected() === true) {
            return this.browser;
        }
        this.browser = await chromium.launch(LAUNCH);
        return this.browser;
    }
}
