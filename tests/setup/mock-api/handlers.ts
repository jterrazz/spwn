import type { ConversationScript } from "./types.js";

// Pre-built conversation scripts for common test scenarios

/** Agent reads a file and responds */
export const readFile = (path: string): ConversationScript => () => ({
  toolCalls: [{ name: "Read", input: { file_path: path } }],
});

/** Agent writes a file */
export const writeFile = (
  path: string,
  content: string,
): ConversationScript => () => ({
  toolCalls: [{ name: "Write", input: { file_path: path, content } }],
});

/** Agent runs a bash command */
export const runBash = (command: string): ConversationScript => () => ({
  toolCalls: [{ name: "Bash", input: { command } }],
});

/** Agent just responds with text (no tool calls) */
export const respond = (text: string): ConversationScript => () => ({
  text,
  stopReason: "end_turn",
});

/** Agent does nothing (immediate end) */
export const noop: ConversationScript = () => ({
  text: "Nothing to do.",
  stopReason: "end_turn",
});
