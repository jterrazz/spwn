/**
 * Types for the ACP Live Activity system.
 * Maps Claude Code stream-json events to renderable UI blocks.
 */

// ── Activity blocks (what the UI renders) ──

export type ActivityBlock =
  | ThinkingBlock
  | TextBlock
  | ToolUseBlock
  | ToolResultBlock
  | ErrorBlock
  | StatusBlock;

export interface ThinkingBlock {
  type: "thinking";
  content: string;
}

export interface TextBlock {
  type: "text";
  content: string;
}

export interface ToolUseBlock {
  type: "tool_use";
  tool: string;
  input: Record<string, unknown>;
  id: string;
}

export interface ToolResultBlock {
  type: "tool_result";
  id: string;
  content: string;
  isError: boolean;
}

export interface ErrorBlock {
  type: "error";
  content: string;
}

export interface StatusBlock {
  type: "status";
  status: "thinking" | "tool_calling" | "responding" | "done";
  tool?: string;
}

// ── Chat message with activity blocks ──

export interface ActivityMessage {
  role: "user" | "agent";
  blocks: ActivityBlock[];
  timestamp: Date;
  /** Cost in USD (from result event) */
  cost?: number;
  /** Duration in ms (from result event) */
  duration?: number;
}

// ── Claude Code stream-json event types ──

export interface StreamJsonEvent {
  type: "system" | "assistant" | "user" | "result" | "stream_event";
  subtype?: string;
  [key: string]: unknown;
}

/**
 * Parse a Claude Code stream-json line into ActivityBlocks.
 * Returns blocks to append to the current message, or null to skip.
 */
export function parseStreamEvent(event: StreamJsonEvent): ActivityBlock[] | null {
  switch (event.type) {
    case "assistant":
      return parseAssistantEvent(event);

    case "user":
      return parseUserEvent(event);

    case "result":
      return parseResultEvent(event);

    default:
      return null;
  }
}

function parseAssistantEvent(event: StreamJsonEvent): ActivityBlock[] | null {
  const message = event.message as {
    content?: Array<{
      type: string;
      text?: string;
      name?: string;
      id?: string;
      input?: Record<string, unknown>;
    }>;
  } | undefined;

  if (!message?.content) return null;

  const blocks: ActivityBlock[] = [];

  for (const block of message.content) {
    if (block.type === "thinking" && block.text) {
      blocks.push({ type: "thinking", content: block.text });
    } else if (block.type === "text" && block.text) {
      blocks.push({ type: "text", content: block.text });
    } else if (block.type === "tool_use" && block.name && block.id) {
      blocks.push({
        type: "tool_use",
        tool: block.name,
        input: block.input || {},
        id: block.id,
      });
    }
  }

  return blocks.length > 0 ? blocks : null;
}

function parseUserEvent(event: StreamJsonEvent): ActivityBlock[] | null {
  const message = event.message as {
    content?: Array<{
      type: string;
      content?: string;
      tool_use_id?: string;
      is_error?: boolean;
    }>;
  } | undefined;

  if (!message?.content) return null;

  const blocks: ActivityBlock[] = [];

  for (const block of message.content) {
    if (block.type === "tool_result" && block.tool_use_id) {
      blocks.push({
        type: "tool_result",
        id: block.tool_use_id,
        content: typeof block.content === "string" ? block.content : JSON.stringify(block.content),
        isError: block.is_error || false,
      });
    }
  }

  return blocks.length > 0 ? blocks : null;
}

function parseResultEvent(event: StreamJsonEvent): ActivityBlock[] | null {
  const blocks: ActivityBlock[] = [];

  // If there's a result text and we haven't already rendered it via assistant events
  const resultText = event.result as string | undefined;
  if (resultText && event.subtype === "success") {
    // The result text is the final response — we may have already streamed it
    // Only add if no assistant text blocks were received (fallback)
  }

  if (event.subtype === "error") {
    const errorText = (event.result as string) || "Unknown error";
    blocks.push({ type: "error", content: errorText });
  }

  return blocks.length > 0 ? blocks : null;
}

/**
 * Deduplicate blocks from assistant events.
 * Claude Code emits cumulative assistant events (each contains all content so far).
 * We need to diff against previous blocks to extract only new content.
 */
export function deduplicateBlocks(
  existingBlocks: ActivityBlock[],
  newBlocks: ActivityBlock[],
): ActivityBlock[] {
  // Find blocks in newBlocks that don't exist in existingBlocks
  const added: ActivityBlock[] = [];

  for (const block of newBlocks) {
    if (block.type === "tool_use") {
      // Tool use blocks are unique by ID
      const exists = existingBlocks.some(
        (b) => b.type === "tool_use" && b.id === block.id
      );
      if (!exists) added.push(block);
    } else if (block.type === "text") {
      // Text blocks: check if this is new or an extension of existing
      const lastExistingText = [...existingBlocks]
        .reverse()
        .find((b) => b.type === "text");
      if (!lastExistingText) {
        added.push(block);
      } else if (block.content !== (lastExistingText as TextBlock).content) {
        // Replace the last text block with the updated one
        // Return a special marker — the caller should update, not append
      }
    } else if (block.type === "thinking") {
      const lastExistingThinking = [...existingBlocks]
        .reverse()
        .find((b) => b.type === "thinking");
      if (!lastExistingThinking) {
        added.push(block);
      }
    } else {
      added.push(block);
    }
  }

  return added;
}
