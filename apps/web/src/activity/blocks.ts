import { fieldOf, listOf, recordOf, textOf } from '../api/fields';

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
            return nonEmpty(listOf(event.message, 'content').flatMap(assistantBlockOf));
        }
        case 'user': {
            return nonEmpty(listOf(event.message, 'content').flatMap(toolResultOf));
        }
        case 'result': {
            return event.subtype === 'error' ? [{ content: errorOf(event), type: 'error' }] : null;
        }
        case 'item.completed': {
            const text = textOf(event.item, 'text');
            return text === '' ? null : [{ content: text, type: 'text' }];
        }
        default: {
            return null;
        }
    }
}

function errorOf(event: StreamJsonEvent): string {
    const error = textOf(event, 'result');
    return error === '' ? 'Unknown error' : error;
}

function nonEmpty(blocks: ActivityBlock[]): ActivityBlock[] | null {
    return blocks.length > 0 ? blocks : null;
}

function assistantBlockOf(block: unknown): ActivityBlock[] {
    const type = textOf(block, 'type');
    const text = textOf(block, 'text');
    if ((type === 'thinking' || type === 'text') && text !== '') {
        return [{ content: text, type }];
    }
    const tool = textOf(block, 'name');
    const id = textOf(block, 'id');
    if (type === 'tool_use' && tool !== '' && id !== '') {
        return [{ id, input: recordOf(block, 'input'), tool, type: 'tool_use' }];
    }
    return [];
}

function toolResultOf(block: unknown): ActivityBlock[] {
    const id = textOf(block, 'tool_use_id');
    if (textOf(block, 'type') !== 'tool_result' || id === '') {
        return [];
    }
    const content = fieldOf(block, 'content');
    return [
        {
            content: typeof content === 'string' ? content : JSON.stringify(content),
            id,
            isError: fieldOf(block, 'is_error') === true,
            type: 'tool_result',
        },
    ];
}

function lastOfType(
    blocks: ActivityBlock[],
    type: ActivityBlock['type'],
): ActivityBlock | undefined {
    return blocks.findLast((block) => block.type === type);
}

/** The blocks of a new event that are not already shown. Claude Code repeats every assistant event cumulatively, so a tool call is new by its id, and text and thinking are new only while the turn has shown none of their kind. */
export function deduplicateBlocks(
    existingBlocks: ActivityBlock[],
    newBlocks: ActivityBlock[],
): ActivityBlock[] {
    return newBlocks.filter((block) => {
        if (block.type === 'tool_use') {
            return !existingBlocks.some(
                (shown) => shown.type === 'tool_use' && shown.id === block.id,
            );
        }
        if (block.type === 'text' || block.type === 'thinking') {
            return lastOfType(existingBlocks, block.type) === undefined;
        }
        return true;
    });
}
