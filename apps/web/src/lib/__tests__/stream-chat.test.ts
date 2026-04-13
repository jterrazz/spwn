import { describe, test, expect, vi, beforeEach } from "vitest";
import { streamChat, type StreamChatOptions } from "../stream-chat";

// ── Helpers ──

/** Build a ReadableStream from an array of string chunks. */
function mockStream(chunks: string[]): ReadableStream<Uint8Array> {
  const encoder = new TextEncoder();
  return new ReadableStream({
    start(controller) {
      for (const chunk of chunks) {
        controller.enqueue(encoder.encode(chunk));
      }
      controller.close();
    },
  });
}

/** Create a minimal mock Response with the given body, status and content-type. */
function mockResponse(
  body: ReadableStream<Uint8Array> | null,
  status = 200,
  contentType = "text/event-stream",
): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    headers: new Headers({ "content-type": contentType }),
    body,
    json: async () => {
      if (!body) return {};
      const reader = body.getReader();
      const decoder = new TextDecoder();
      let text = "";
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        text += decoder.decode(value, { stream: true });
      }
      return JSON.parse(text);
    },
    text: async () => {
      if (!body) return "";
      const reader = body.getReader();
      const decoder = new TextDecoder();
      let text = "";
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        text += decoder.decode(value, { stream: true });
      }
      return text;
    },
  } as unknown as Response;
}

/** Collect blocks/errors/done calls from streamChat. */
function createCallbacks() {
  const blocks: unknown[][] = [];
  const texts: string[] = [];
  const errors: string[] = [];
  const dones: Array<{ cost?: number; duration?: number }> = [];

  return {
    blocks,
    texts,
    errors,
    dones,
    onBlocks: (b: unknown[]) => blocks.push(b),
    onText: (t: string) => texts.push(t),
    onDone: (meta: { cost?: number; duration?: number }) => dones.push(meta),
    onError: (e: string) => errors.push(e),
  };
}

// ── Tests ──

describe("streamChat", () => {
  let fetchSpy: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchSpy = vi.fn();
    vi.stubGlobal("fetch", fetchSpy);
  });

  // ──────────────────────────────────────────────
  // SSE stream with structured assistant events
  // ──────────────────────────────────────────────
  test("parses SSE stream with assistant text events", async () => {
    const sseData = [
      'data: {"type":"assistant","message":{"content":[{"type":"text","text":"hello world"}]}}\n\n',
      "data: [DONE]\n\n",
    ];

    fetchSpy.mockResolvedValueOnce(mockResponse(mockStream(sseData)));

    const cb = createCallbacks();
    await streamChat({
      url: "http://localhost:9999/talk",
      body: { message: "hi" },
      ...cb,
    });

    expect(cb.errors).toHaveLength(0);
    expect(cb.dones).toHaveLength(1);
    expect(cb.blocks.length).toBeGreaterThan(0);

    // The first batch should contain a text block with "hello world"
    const allBlocks = cb.blocks.flat();
    const textBlock = allBlocks.find(
      (b: any) => b.type === "text" && b.content === "hello world",
    );
    expect(textBlock).toBeDefined();
  });

  // ──────────────────────────────────────────────
  // SSE stream with tool_use events
  // ──────────────────────────────────────────────
  test("parses SSE stream with tool_use events", async () => {
    const sseData = [
      'data: {"type":"assistant","message":{"content":[{"type":"tool_use","name":"bash","id":"t1","input":{"command":"ls"}}]}}\n\n',
      "data: [DONE]\n\n",
    ];

    fetchSpy.mockResolvedValueOnce(mockResponse(mockStream(sseData)));

    const cb = createCallbacks();
    await streamChat({
      url: "http://localhost:9999/talk",
      body: { message: "list files" },
      ...cb,
    });

    expect(cb.errors).toHaveLength(0);
    const allBlocks = cb.blocks.flat();
    const toolBlock = allBlocks.find(
      (b: any) => b.type === "tool_use" && b.tool === "bash",
    );
    expect(toolBlock).toBeDefined();
  });

  // ──────────────────────────────────────────────
  // SSE stream with result event containing cost/duration
  // ──────────────────────────────────────────────
  test("extracts cost and duration from result event", async () => {
    const sseData = [
      'data: {"type":"assistant","message":{"content":[{"type":"text","text":"done"}]}}\n\n',
      'data: {"type":"result","subtype":"success","total_cost_usd":0.05,"duration_ms":1234}\n\n',
      "data: [DONE]\n\n",
    ];

    fetchSpy.mockResolvedValueOnce(mockResponse(mockStream(sseData)));

    const cb = createCallbacks();
    await streamChat({
      url: "http://localhost:9999/talk",
      body: { message: "hi" },
      ...cb,
    });

    expect(cb.dones).toHaveLength(1);
    expect(cb.dones[0].cost).toBe(0.05);
    expect(cb.dones[0].duration).toBe(1234);
  });

  // ──────────────────────────────────────────────
  // Plain text stream (non-SSE, no "data: " prefix)
  // ──────────────────────────────────────────────
  test("handles plain text stream without SSE prefix", async () => {
    const plainChunks = ["Hello, this is a plain response.\n", "Second line.\n"];

    fetchSpy.mockResolvedValueOnce(
      mockResponse(mockStream(plainChunks), 200, "text/plain"),
    );

    const cb = createCallbacks();
    await streamChat({
      url: "http://localhost:9999/talk",
      body: { message: "hi" },
      ...cb,
    });

    expect(cb.errors).toHaveLength(0);
    expect(cb.dones).toHaveLength(1);

    // Plain text should be emitted as text blocks or via onText
    const allBlocks = cb.blocks.flat();
    const allTexts = cb.texts;
    expect(allBlocks.length + allTexts.length).toBeGreaterThan(0);
  });

  // ──────────────────────────────────────────────
  // JSON response fallback (non-streaming)
  // ──────────────────────────────────────────────
  test("handles JSON response fallback", async () => {
    const jsonBody = JSON.stringify({
      response: "This is a JSON fallback response",
    });

    fetchSpy.mockResolvedValueOnce(
      mockResponse(mockStream([jsonBody]), 200, "application/json"),
    );

    const cb = createCallbacks();
    await streamChat({
      url: "http://localhost:9999/talk",
      body: { message: "hi" },
      ...cb,
    });

    expect(cb.errors).toHaveLength(0);
    expect(cb.dones).toHaveLength(1);

    const allBlocks = cb.blocks.flat();
    const textBlock = allBlocks.find(
      (b: any) =>
        b.type === "text" && b.content.includes("JSON fallback response"),
    );
    expect(textBlock).toBeDefined();
  });

  // ──────────────────────────────────────────────
  // Network error (fetch throws)
  // ──────────────────────────────────────────────
  test("handles network error", async () => {
    fetchSpy.mockRejectedValueOnce(new Error("ECONNREFUSED"));

    const cb = createCallbacks();
    await streamChat({
      url: "http://localhost:9999/talk",
      body: { message: "hi" },
      ...cb,
    });

    expect(cb.errors).toHaveLength(1);
    expect(cb.errors[0]).toContain("ECONNREFUSED");
    expect(cb.dones).toHaveLength(0);
  });

  // ──────────────────────────────────────────────
  // HTTP error response (non-2xx)
  // ──────────────────────────────────────────────
  test("handles HTTP error response", async () => {
    const errorBody = JSON.stringify({ error: "Internal server error" });
    const errorResponse = {
      ok: false,
      status: 500,
      headers: new Headers({ "content-type": "application/json" }),
      json: async () => JSON.parse(errorBody),
      text: async () => errorBody,
      body: null,
    } as unknown as Response;

    fetchSpy.mockResolvedValueOnce(errorResponse);

    const cb = createCallbacks();
    await streamChat({
      url: "http://localhost:9999/talk",
      body: { message: "hi" },
      ...cb,
    });

    expect(cb.errors).toHaveLength(1);
    expect(cb.errors[0]).toContain("Internal server error");
  });

  // ──────────────────────────────────────────────
  // Abort signal cancellation
  // ──────────────────────────────────────────────
  test("handles abort signal cancellation", async () => {
    const controller = new AbortController();
    controller.abort();

    fetchSpy.mockRejectedValueOnce(new DOMException("Aborted", "AbortError"));

    const cb = createCallbacks();
    await streamChat({
      url: "http://localhost:9999/talk",
      body: { message: "hi" },
      signal: controller.signal,
      ...cb,
    });

    expect(cb.errors).toHaveLength(1);
    expect(cb.errors[0]).toBe("Request cancelled");
  });

  // ──────────────────────────────────────────────
  // Fallback URL on primary failure
  // ──────────────────────────────────────────────
  test("falls back to fallbackUrl when primary fails", async () => {
    // Primary fails
    fetchSpy.mockRejectedValueOnce(new Error("ECONNREFUSED"));

    // Fallback succeeds
    const sseData = [
      'data: {"type":"assistant","message":{"content":[{"type":"text","text":"fallback response"}]}}\n\n',
      "data: [DONE]\n\n",
    ];
    fetchSpy.mockResolvedValueOnce(mockResponse(mockStream(sseData)));

    const cb = createCallbacks();
    await streamChat({
      url: "http://localhost:9999/talk",
      fallbackUrl: "http://localhost:3000/api/architect/talk",
      body: { message: "hi" },
      ...cb,
    });

    expect(cb.errors).toHaveLength(0);
    expect(cb.dones).toHaveLength(1);
    expect(fetchSpy).toHaveBeenCalledTimes(2);
  });

  // ──────────────────────────────────────────────
  // Fallback also fails
  // ──────────────────────────────────────────────
  test("reports primary error when both primary and fallback fail", async () => {
    fetchSpy.mockRejectedValueOnce(new Error("Primary down"));
    fetchSpy.mockRejectedValueOnce(new Error("Fallback down"));

    const cb = createCallbacks();
    await streamChat({
      url: "http://localhost:9999/talk",
      fallbackUrl: "http://localhost:3000/api/architect/talk",
      body: { message: "hi" },
      ...cb,
    });

    expect(cb.errors).toHaveLength(1);
    expect(cb.errors[0]).toContain("Primary down");
  });

  // ──────────────────────────────────────────────
  // Response with no body reader falls back to text
  // ──────────────────────────────────────────────
  test("handles response with null body", async () => {
    const response = {
      ok: true,
      status: 200,
      headers: new Headers({ "content-type": "text/plain" }),
      body: null,
      text: async () => "plain text fallback",
    } as unknown as Response;

    fetchSpy.mockResolvedValueOnce(response);

    const cb = createCallbacks();
    await streamChat({
      url: "http://localhost:9999/talk",
      body: { message: "hi" },
      ...cb,
    });

    expect(cb.errors).toHaveLength(0);
    expect(cb.dones).toHaveLength(1);

    const allBlocks = cb.blocks.flat();
    const textBlock = allBlocks.find(
      (b: any) => b.type === "text" && b.content.includes("plain text"),
    );
    expect(textBlock).toBeDefined();
  });

  // ──────────────────────────────────────────────
  // SSE stream with thinking events
  // ──────────────────────────────────────────────
  test("parses thinking events from SSE stream", async () => {
    const sseData = [
      'data: {"type":"assistant","message":{"content":[{"type":"thinking","text":"Let me think about this..."}]}}\n\n',
      'data: {"type":"assistant","message":{"content":[{"type":"text","text":"Here is my answer."}]}}\n\n',
      "data: [DONE]\n\n",
    ];

    fetchSpy.mockResolvedValueOnce(mockResponse(mockStream(sseData)));

    const cb = createCallbacks();
    await streamChat({
      url: "http://localhost:9999/talk",
      body: { message: "hi" },
      ...cb,
    });

    const allBlocks = cb.blocks.flat();
    const thinkingBlock = allBlocks.find(
      (b: any) => b.type === "thinking",
    );
    expect(thinkingBlock).toBeDefined();
  });

  // ──────────────────────────────────────────────
  // SSE with result error subtype
  // ──────────────────────────────────────────────
  test("parses result error event into error block", async () => {
    const sseData = [
      'data: {"type":"result","subtype":"error","result":"Rate limit exceeded"}\n\n',
      "data: [DONE]\n\n",
    ];

    fetchSpy.mockResolvedValueOnce(mockResponse(mockStream(sseData)));

    const cb = createCallbacks();
    await streamChat({
      url: "http://localhost:9999/talk",
      body: { message: "hi" },
      ...cb,
    });

    const allBlocks = cb.blocks.flat();
    const errorBlock = allBlocks.find(
      (b: any) => b.type === "error" && b.content.includes("Rate limit"),
    );
    expect(errorBlock).toBeDefined();
  });
});
