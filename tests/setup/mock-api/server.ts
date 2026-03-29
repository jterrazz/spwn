import { Hono } from "hono";
import { serve } from "@hono/node-server";
import type {
  MockApiCall,
  ConversationScript,
  ScriptedResponse,
} from "./types.js";

export class MockApiServer {
  private app: Hono;
  private server: ReturnType<typeof serve> | null = null;
  private script: ConversationScript | null = null;
  calls: MockApiCall[] = [];
  port: number;

  constructor(port = 9999) {
    this.port = port;
    this.app = new Hono();
    this.setupRoutes();
  }

  private setupRoutes() {
    // Anthropic Messages API
    this.app.post("/v1/messages", async (c) => {
      const body = await c.req.json();

      const call: MockApiCall = {
        timestamp: new Date(),
        model: body.model,
        messages: body.messages,
        tools: (body.tools || []).map((t: { name: string }) => t.name),
      };
      this.calls.push(call);

      // Get scripted response
      const response = this.script
        ? this.script(body.messages)
        : { text: "Mock response", stopReason: "end_turn" as const };

      // Return Anthropic-format response
      return c.json({
        id: `msg_${Date.now()}`,
        type: "message",
        role: "assistant",
        model: body.model || "claude-sonnet-4-6",
        content: this.buildContent(response),
        stop_reason: response.toolCalls?.length ? "tool_use" : "end_turn",
        usage: { input_tokens: 100, output_tokens: 50 },
      });
    });
  }

  private buildContent(response: ScriptedResponse) {
    const content: Record<string, unknown>[] = [];

    if (response.text) {
      content.push({ type: "text", text: response.text });
    }

    if (response.toolCalls) {
      for (const call of response.toolCalls) {
        content.push({
          type: "tool_use",
          id: `toolu_${Date.now()}_${Math.random().toString(36).slice(2)}`,
          name: call.name,
          input: call.input,
        });
      }
    }

    if (content.length === 0) {
      content.push({ type: "text", text: "Done." });
    }

    return content;
  }

  onChat(script: ConversationScript) {
    this.script = script;
  }

  reset() {
    this.calls = [];
    this.script = null;
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
