/** Types for the ACP Live Activity system. Maps Claude Code stream-json events to renderable UI blocks. */

export type ActivityBlock =
    | ErrorBlock
    | StatusBlock
    | TextBlock
    | ThinkingBlock
    | ToolResultBlock
    | ToolUseBlock;

export interface ThinkingBlock {
    type: 'thinking';
    content: string;
}

export interface TextBlock {
    type: 'text';
    content: string;
}

export interface ToolUseBlock {
    type: 'tool_use';
    tool: string;
    input: Record<string, unknown>;
    id: string;
}

export interface ToolResultBlock {
    type: 'tool_result';
    id: string;
    content: string;
    isError: boolean;
}

export interface ErrorBlock {
    type: 'error';
    content: string;
}

export interface StatusBlock {
    type: 'status';
    status: 'done' | 'responding' | 'thinking' | 'tool_calling';
    tool?: string;
}

export interface ActivityMessage {
    role: 'agent' | 'user';
    blocks: ActivityBlock[];
    timestamp: Date;
    /** Cost in USD (from result event) */
    cost?: number | undefined;
    /** Duration in ms (from result event) */
    duration?: number | undefined;
}

export interface StreamJsonEvent {
    type: string;
    subtype?: string;
    [key: string]: unknown;
}

/** Parse a stream-json line into ActivityBlocks. Handles both Claude Code events (assistant/result) and Codex JSONL (item.completed/turn.completed). */
export function parseStreamEvent(event: StreamJsonEvent): ActivityBlock[] | null {
    switch (event.type) {
        case 'assistant': {
            return parseAssistantEvent(event);
        }
        case 'user': {
            return parseUserEvent(event);
        }
        case 'result': {
            return parseResultEvent(event);
        }

        case 'item.completed': {
            const item = event.item as undefined | { type?: string; text?: string };
            if (item?.text !== undefined && item?.text !== '') {
                return [{ type: 'text', content: item.text }];
            }
            return null;
        }
        case 'turn.completed': {
            return null;
        }

        default: {
            return null;
        }
    }
}

function parseAssistantEvent(event: StreamJsonEvent): ActivityBlock[] | null {
    const message = event.message as
        | undefined
        | {
              content?: {
                  type: string;
                  text?: string;
                  name?: string;
                  id?: string;
                  input?: Record<string, unknown>;
              }[];
          };

    if (!message?.content) {
        return null;
    }

    const blocks: ActivityBlock[] = [];

    for (const block of message.content) {
        if (block.type === 'thinking' && block.text !== undefined && block.text !== '') {
            blocks.push({ type: 'thinking', content: block.text });
        } else if (block.type === 'text' && block.text !== undefined && block.text !== '') {
            blocks.push({ type: 'text', content: block.text });
        } else if (
            block.type === 'tool_use' &&
            block.name !== undefined &&
            block.name !== '' &&
            block.id !== undefined &&
            block.id !== ''
        ) {
            blocks.push({
                type: 'tool_use',
                tool: block.name,
                input: block.input || {},
                id: block.id,
            });
        }
    }

    return blocks.length > 0 ? blocks : null;
}

function parseUserEvent(event: StreamJsonEvent): ActivityBlock[] | null {
    const message = event.message as
        | undefined
        | {
              content?: {
                  type: string;
                  content?: string;
                  tool_use_id?: string;
                  is_error?: boolean;
              }[];
          };

    if (!message?.content) {
        return null;
    }

    const blocks: ActivityBlock[] = [];

    for (const block of message.content) {
        if (
            block.type === 'tool_result' &&
            block.tool_use_id !== undefined &&
            block.tool_use_id !== ''
        ) {
            blocks.push({
                type: 'tool_result',
                id: block.tool_use_id,
                content:
                    typeof block.content === 'string'
                        ? block.content
                        : JSON.stringify(block.content),
                isError: block.is_error === true || false,
            });
        }
    }

    return blocks.length > 0 ? blocks : null;
}

function parseResultEvent(event: StreamJsonEvent): ActivityBlock[] | null {
    const blocks: ActivityBlock[] = [];

    const resultText = event.result as string | undefined;
    if (resultText !== undefined && resultText !== '' && event.subtype === 'success') {
    }

    if (event.subtype === 'error') {
        const errorText = (event.result as string) || 'Unknown error';
        blocks.push({ type: 'error', content: errorText });
    }

    return blocks.length > 0 ? blocks : null;
}

/** Deduplicate blocks from assistant events. Claude Code emits cumulative assistant events (each contains all content so far). We need to diff against previous blocks to extract only new content. */
export function deduplicateBlocks(
    existingBlocks: ActivityBlock[],
    newBlocks: ActivityBlock[],
): ActivityBlock[] {
    const added: ActivityBlock[] = [];

    for (const block of newBlocks) {
        if (block.type === 'tool_use') {
            const exists = existingBlocks.some((b) => b.type === 'tool_use' && b.id === block.id);
            if (!exists) {
                added.push(block);
            }
        } else if (block.type === 'text') {
            const lastExistingText = [...existingBlocks]
                .toReversed()
                .find((b) => b.type === 'text');
            if (!lastExistingText) {
                added.push(block);
            } else if (block.content !== lastExistingText.content) {
            }
        } else if (block.type === 'thinking') {
            const lastExistingThinking = [...existingBlocks]
                .toReversed()
                .find((b) => b.type === 'thinking');
            if (!lastExistingThinking) {
                added.push(block);
            }
        } else {
            added.push(block);
        }
    }

    return added;
}
