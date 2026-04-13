/**
 * Shared utility for consuming streaming chat responses from the Go API.
 * Parses Claude Code stream-json events into ActivityBlocks.
 */

import type { ActivityBlock, StreamJsonEvent } from "./activity-types";
import { parseStreamEvent, deduplicateBlocks } from "./activity-types";

export interface StreamChatOptions {
  /** Primary URL (Go API) */
  url: string;
  /** @deprecated No longer used — Go API is the sole backend */
  fallbackUrl?: string;
  /** POST body */
  body: Record<string, unknown>;
  /** Called with new ActivityBlocks as they arrive */
  onBlocks: (blocks: ActivityBlock[]) => void;
  /** Called with raw text chunks (for non-structured responses) */
  onText?: (text: string) => void;
  /** Called when the stream completes */
  onDone: (meta: { cost?: number; duration?: number }) => void;
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
): Promise<{ cost?: number; duration?: number }> {
  const contentType = res.headers.get("content-type") || "";

  // If JSON response (non-streaming fallback), extract text
  if (contentType.includes("application/json")) {
    const data = await res.json();
    const text = data.response || data.output || data.message || data.error || "No response";
    onBlocks([{ type: "text", content: text }]);
    return {};
  }

  const reader = res.body?.getReader();
  if (!reader) {
    const text = await res.text();
    onBlocks([{ type: "text", content: text }]);
    return {};
  }

  const decoder = new TextDecoder();
  let buffer = "";
  let meta: { cost?: number; duration?: number } = {};
  let allBlocks: ActivityBlock[] = [];

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;

    buffer += decoder.decode(value, { stream: true });

    // Process complete lines
    const lines = buffer.split("\n");
    buffer = lines.pop() || ""; // Keep incomplete line in buffer

    for (const line of lines) {
      const trimmed = line.trim();

      // SSE format: "data: {...}"
      if (trimmed.startsWith("data: ")) {
        const data = trimmed.slice(6);

        if (data === "[DONE]") continue;

        try {
          const event: StreamJsonEvent = JSON.parse(data);

          // Extract cost/duration from result events (Claude + Codex)
          if (event.type === "result") {
            meta.cost = event.total_cost_usd as number | undefined;
            meta.duration = event.duration_ms as number | undefined;
          }
          if (event.type === "turn.completed") {
            const usage = event.usage as { input_tokens?: number; output_tokens?: number } | undefined;
            if (usage) {
              // Codex doesn't report cost directly — estimate from token count
              meta.duration = undefined;
            }
          }

          const newBlocks = parseStreamEvent(event);
          if (newBlocks) {
            const deduplicated = deduplicateBlocks(allBlocks, newBlocks);
            if (deduplicated.length > 0) {
              allBlocks = [...allBlocks, ...deduplicated];
              onBlocks(deduplicated);
            }
          }
        } catch {
          // Not valid JSON — treat as plain text
          if (onText) onText(data);
          else onBlocks([{ type: "text", content: data }]);
        }
      } else if (trimmed && !trimmed.startsWith(":")) {
        // Plain text line (non-SSE format, e.g. from Next.js fallback)
        try {
          const event: StreamJsonEvent = JSON.parse(trimmed);
          const newBlocks = parseStreamEvent(event);
          if (newBlocks) {
            const deduplicated = deduplicateBlocks(allBlocks, newBlocks);
            if (deduplicated.length > 0) {
              allBlocks = [...allBlocks, ...deduplicated];
              onBlocks(deduplicated);
            }
          }
          if (event.type === "result") {
            meta.cost = event.total_cost_usd as number | undefined;
            meta.duration = event.duration_ms as number | undefined;
          }
        } catch {
          // Plain text
          if (onText) onText(trimmed + "\n");
          else onBlocks([{ type: "text", content: trimmed + "\n" }]);
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
  timeout = 120000,
}: StreamChatOptions): Promise<void> {
  const fetchOptions: RequestInit = {
    method: "POST",
    headers: { "Content-Type": "application/json" },
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
    if (signal?.aborted) {
      onError("Request cancelled");
      return;
    }

    if (fallbackUrl) {
      try {
        const res = await fetch(fallbackUrl, {
          ...fetchOptions,
          signal: AbortSignal.timeout(timeout),
        });
        if (!res.ok) {
          const data = await res.json().catch(() => ({}));
          throw new Error(data.error || `HTTP ${res.status}`);
        }
        const meta = await consumeSSEStream(res, onBlocks, onText);
        onDone(meta);
      } catch {
        onError(primaryError instanceof Error ? primaryError.message : "Failed to connect");
      }
    } else {
      onError(primaryError instanceof Error ? primaryError.message : "Failed to connect");
    }
  }
}
