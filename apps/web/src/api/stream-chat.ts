import { deduplicateBlocks, parseStreamEvent } from '../activity/blocks';
import type { ActivityBlock, StreamJsonEvent } from '../activity/blocks';

export interface StreamChatOptions {
    /** Primary URL (Go API) */
    url: string;
    /** @deprecated No longer used - Go API is the sole backend */
    fallbackUrl?: string;
    /** POST body */
    body: Record<string, unknown>;
    /** Called with new ActivityBlocks as they arrive */
    onBlocks: (blocks: ActivityBlock[]) => void;
    /** Called with raw text chunks (for non-structured responses) */
    onText?: (text: string) => void;
    /** Called when the stream completes */
    onDone: (meta: { cost?: number | undefined; duration?: number | undefined }) => void;
    /** Called on error */
    onError: (error: string) => void;
    /** AbortSignal for cancellation */
    signal?: AbortSignal;
    /** Timeout in ms (default 120s) */
    timeout?: number;
}

async function consumeSSEStream(
    res: Response,
    onBlocks: (blocks: ActivityBlock[]) => void,
    onText?: (text: string) => void,
): Promise<{ cost?: number | undefined; duration?: number | undefined }> {
    const contentType = res.headers.get('content-type') ?? '';

    if (contentType.includes('application/json')) {
        const data = await res.json();
        const text = data.response || data.output || data.message || data.error || 'No response';
        onBlocks([{ type: 'text', content: text }]);
        return {};
    }

    const reader = res.body?.getReader();
    if (!reader) {
        const text = await res.text();
        onBlocks([{ type: 'text', content: text }]);
        return {};
    }

    const decoder = new TextDecoder();
    let buffer = '';
    const meta: { cost?: number | undefined; duration?: number | undefined } = {};
    const allBlocks: ActivityBlock[] = [];

    while (true) {
        const { done, value } = await reader.read();
        if (done) {
            break;
        }

        buffer += decoder.decode(value, { stream: true });

        const lines = buffer.split('\n');
        buffer = lines.pop() ?? '';

        for (const line of lines) {
            const trimmed = line.trim();

            if (trimmed.startsWith('data: ')) {
                const data = trimmed.slice(6);

                if (data === '[DONE]') {
                    continue;
                }

                try {
                    const event: StreamJsonEvent = JSON.parse(data);

                    if (event.type === 'result') {
                        meta.cost = event.total_cost_usd as number | undefined;
                        meta.duration = event.duration_ms as number | undefined;
                    }
                    if (event.type === 'turn.completed') {
                        const usage = event.usage as
                            | undefined
                            | { input_tokens?: number; output_tokens?: number };
                        if (usage) {
                            meta.duration = undefined;
                        }
                    }

                    const newBlocks = parseStreamEvent(event);
                    if (newBlocks) {
                        const deduplicated = deduplicateBlocks(allBlocks, newBlocks);
                        if (deduplicated.length > 0) {
                            allBlocks.push(...deduplicated);
                            onBlocks(deduplicated);
                        }
                    }
                } catch {
                    if (onText) {
                        onText(data);
                    } else {
                        onBlocks([{ type: 'text', content: data }]);
                    }
                }
            } else if (trimmed && !trimmed.startsWith(':')) {
                try {
                    const event: StreamJsonEvent = JSON.parse(trimmed);
                    const newBlocks = parseStreamEvent(event);
                    if (newBlocks) {
                        const deduplicated = deduplicateBlocks(allBlocks, newBlocks);
                        if (deduplicated.length > 0) {
                            allBlocks.push(...deduplicated);
                            onBlocks(deduplicated);
                        }
                    }
                    if (event.type === 'result') {
                        meta.cost = event.total_cost_usd as number | undefined;
                        meta.duration = event.duration_ms as number | undefined;
                    }
                } catch {
                    if (onText) {
                        onText(`${trimmed}\n`);
                    } else {
                        onBlocks([{ type: 'text', content: `${trimmed}\n` }]);
                    }
                }
            }
        }
    }

    return meta;
}

export async function streamChat({
    url,
    fallbackUrl,
    body,
    onBlocks,
    onText,
    onDone,
    onError,
    signal,
    timeout = 120_000,
}: StreamChatOptions): Promise<void> {
    const fetchOptions: RequestInit = {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
        signal: signal || AbortSignal.timeout(timeout),
    };

    try {
        const res = await fetch(url, fetchOptions);
        if (!res.ok) {
            const data = await res.json().catch(() => ({}));
            throw new Error(data.error || `HTTP ${res.status}`);
        }
        const meta = await consumeSSEStream(res, onBlocks, onText);
        onDone(meta);
    } catch (primaryError) {
        if (signal?.aborted === true) {
            onError('Request cancelled');
            return;
        }

        if (fallbackUrl === undefined || fallbackUrl === '') {
            onError(primaryError instanceof Error ? primaryError.message : 'Failed to connect');
        } else {
            try {
                const res = await fetch(fallbackUrl, {
                    ...fetchOptions,
                    signal: AbortSignal.timeout(timeout),
                });
                if (!res.ok) {
                    const data = await res.json().catch(() => ({}));
                    throw new Error(data.error || `HTTP ${res.status}`, { cause: primaryError });
                }
                const meta = await consumeSSEStream(res, onBlocks, onText);
                onDone(meta);
            } catch {
                onError(primaryError instanceof Error ? primaryError.message : 'Failed to connect');
            }
        }
    }
}
