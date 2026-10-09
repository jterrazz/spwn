import type { BrowserContext, Page, Response } from 'playwright';

const IDLE_TTL_MS = 5 * 60 * 1000;
export const HARD_TTL_MS = 30 * 60 * 1000;
const KEPT_RESPONSES = 500;
const DROPPED_RESPONSES = 100;

/** A JSON or script response the page received, its body read only when a tool asks for it. */
export type Captured = {
    method: string;
    response: Response;
    status: number;
    ts: number;
    url: string;
};

/** One tool's browser context: its own cookie jar, one page, and the responses that page received. */
export class Session {
    readonly captured: Captured[] = [];
    readonly createdAt = Date.now();
    readonly page: Page;
    private readonly context: BrowserContext;
    private lastUsedAt = Date.now();

    constructor(context: BrowserContext, page: Page) {
        this.context = context;
        this.page = page;
        page.on('response', (response) => {
            const type = response.headers()['content-type']?.toLowerCase();
            if (type?.includes('json') !== true && type?.includes('javascript') !== true) {
                return;
            }
            this.captured.push({
                method: response.request().method(),
                response,
                status: response.status(),
                ts: Date.now(),
                url: response.url(),
            });
            if (this.captured.length > KEPT_RESPONSES) {
                this.captured.splice(0, DROPPED_RESPONSES);
            }
        });
    }

    touch(): void {
        this.lastUsedAt = Date.now();
    }

    expired(now: number): boolean {
        return now - this.lastUsedAt > IDLE_TTL_MS || now - this.createdAt > HARD_TTL_MS;
    }

    async close(): Promise<void> {
        await this.context.close().catch(() => undefined);
    }
}
