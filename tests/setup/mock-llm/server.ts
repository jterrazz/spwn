import { Hono } from "hono";
import { serve } from "@hono/node-server";
import { streamSSE } from "hono/streaming";
import { HELLO_RESPONSE } from "./fixtures.js";

export interface LLMCall {
  timestamp: Date;
  provider: "anthropic" | "openai";
  model: string;
  messages: unknown[];
  stream: boolean;
}

export class MockLLMServer {
  private app: Hono;
  private server: ReturnType<typeof serve> | null = null;
  private anthropicResponse: string = HELLO_RESPONSE;
  private openaiResponse: string = HELLO_RESPONSE;
  calls: LLMCall[] = [];
  port: number;

  constructor(port = 5555) {
    this.port = port;
    this.app = new Hono();
    this.setupRoutes();
  }

  private setupRoutes() {
    // Anthropic Messages API
    this.app.post("/v1/messages", async (c) => {
      const body = await c.req.json();
      const isStream = body.stream === true;

      const call: LLMCall = {
        timestamp: new Date(),
        provider: "anthropic",
        model: body.model ?? "claude-sonnet-4-6",
        messages: body.messages ?? [],
        stream: isStream,
      };
      this.calls.push(call);

      const text = this.anthropicResponse;

      if (isStream) {
        return streamSSE(c, async (stream) => {
          await stream.writeSSE({
            event: "message_start",
            data: JSON.stringify({
              type: "message_start",
              message: {
                id: `msg_${Date.now()}`,
                type: "message",
                role: "assistant",
                model: body.model ?? "claude-sonnet-4-6",
                content: [],
                usage: { input_tokens: 10, output_tokens: 0 },
              },
            }),
          });

          await stream.writeSSE({
            event: "content_block_start",
            data: JSON.stringify({
              type: "content_block_start",
              index: 0,
              content_block: { type: "text", text: "" },
            }),
          });

          await stream.writeSSE({
            event: "content_block_delta",
            data: JSON.stringify({
              type: "content_block_delta",
              index: 0,
              delta: { type: "text_delta", text },
            }),
          });

          await stream.writeSSE({
            event: "content_block_stop",
            data: JSON.stringify({
              type: "content_block_stop",
              index: 0,
            }),
          });

          await stream.writeSSE({
            event: "message_delta",
            data: JSON.stringify({
              type: "message_delta",
              delta: { stop_reason: "end_turn" },
              usage: { output_tokens: 5 },
            }),
          });

          await stream.writeSSE({
            event: "message_stop",
            data: JSON.stringify({ type: "message_stop" }),
          });
        });
      }

      return c.json({
        id: `msg_${Date.now()}`,
        type: "message",
        role: "assistant",
        model: body.model ?? "claude-sonnet-4-6",
        content: [{ type: "text", text }],
        stop_reason: "end_turn",
        usage: { input_tokens: 10, output_tokens: 5 },
      });
    });

    // OpenAI Chat Completions API
    this.app.post("/v1/chat/completions", async (c) => {
      const body = await c.req.json();
      const isStream = body.stream === true;

      const call: LLMCall = {
        timestamp: new Date(),
        provider: "openai",
        model: body.model ?? "gpt-4",
        messages: body.messages ?? [],
        stream: isStream,
      };
      this.calls.push(call);

      const text = this.openaiResponse;

      if (isStream) {
        return streamSSE(c, async (stream) => {
          await stream.writeSSE({
            data: JSON.stringify({
              id: `chatcmpl-${Date.now()}`,
              object: "chat.completion.chunk",
              model: body.model ?? "gpt-4",
              choices: [
                {
                  index: 0,
                  delta: { role: "assistant", content: text },
                  finish_reason: null,
                },
              ],
            }),
          });

          await stream.writeSSE({
            data: JSON.stringify({
              id: `chatcmpl-${Date.now()}`,
              object: "chat.completion.chunk",
              model: body.model ?? "gpt-4",
              choices: [
                {
                  index: 0,
                  delta: {},
                  finish_reason: "stop",
                },
              ],
            }),
          });

          await stream.writeSSE({ data: "[DONE]" });
        });
      }

      return c.json({
        id: `chatcmpl-${Date.now()}`,
        object: "chat.completion",
        model: body.model ?? "gpt-4",
        choices: [
          {
            index: 0,
            message: { role: "assistant", content: text },
            finish_reason: "stop",
          },
        ],
        usage: {
          prompt_tokens: 10,
          completion_tokens: 5,
          total_tokens: 15,
        },
      });
    });

    // Health check
    this.app.get("/health", (c) => c.json({ status: "ok" }));
  }

  /** Set the response text for Anthropic API calls. */
  setAnthropicResponse(text: string) {
    this.anthropicResponse = text;
  }

  /** Set the response text for OpenAI API calls. */
  setOpenAIResponse(text: string) {
    this.openaiResponse = text;
  }

  /** Get calls filtered by provider. */
  callsByProvider(provider: "anthropic" | "openai"): LLMCall[] {
    return this.calls.filter((c) => c.provider === provider);
  }

  /** Reset all recorded calls and restore default responses. */
  reset() {
    this.calls = [];
    this.anthropicResponse = HELLO_RESPONSE;
    this.openaiResponse = HELLO_RESPONSE;
  }

  async start() {
    this.server = serve({ fetch: this.app.fetch, port: this.port });
  }

  async stop() {
    if (this.server) {
      this.server.close();
      this.server = null;
    }
  }
}
