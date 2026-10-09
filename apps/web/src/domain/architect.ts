import type { ActivityBlock } from '../activity/blocks';
import { listOf, numberOf, textOf } from '../api/fields';

export interface ArchitectStatus {
    status: 'running' | 'stopped';
    containerId: null | string;
    uptime: null | string;
    error?: string;
    kpis?: {
        worlds: number;
        agents: number;
        tasksPending: number;
        tasksCompleted: number;
    };
}

export interface ChatMessage {
    role: 'architect' | 'user';
    content: string;
    blocks: ActivityBlock[];
    timestamp: Date;
    error?: boolean | undefined;
    cost?: number | undefined;
    duration?: number | undefined;
}

export const STOPPED_ARCHITECT: ArchitectStatus = {
    containerId: null,
    status: 'stopped',
    uptime: null,
};

function dateOf(stamp: string): Date {
    return stamp === '' ? new Date() : new Date(stamp);
}

export function said(
    role: ChatMessage['role'],
    content: string,
    timestamp = new Date(),
): ChatMessage {
    return { blocks: [{ content, type: 'text' }], content, role, timestamp };
}

export function failed(content: string, shown = content): ChatMessage {
    return {
        blocks: [{ content, type: 'error' }],
        content: shown,
        error: true,
        role: 'architect',
        timestamp: new Date(),
    };
}

function sessionDivider(session: unknown): ChatMessage {
    const startedAt = textOf(session, 'startedAt');
    const when = startedAt === '' ? 'unknown' : new Date(startedAt).toLocaleString();
    return {
        ...said('architect', '── Previous session ──', dateOf(startedAt)),
        content: `── Session ${textOf(session, 'id').slice(0, 8)} · ${when} ──`,
    };
}

function messageOf(entry: unknown, id: string): ChatMessage | undefined {
    const type = textOf(entry, 'type');
    const role = textOf(entry, 'role');
    const timestamp = dateOf(textOf(entry, 'timestamp'));
    if (type === 'text' && (role === 'user' || role === 'assistant')) {
        return said(role === 'user' ? 'user' : 'architect', textOf(entry, 'content'), timestamp);
    }
    if (type === 'tool_use') {
        const tool = textOf(entry, 'toolName') || 'tool';
        return {
            blocks: [{ id, input: {}, tool, type: 'tool_use' }],
            content: `🔧 ${tool}`,
            role: 'architect',
            timestamp,
        };
    }
    return undefined;
}

function priceLastReply(messages: ChatMessage[], entry: unknown): void {
    const cost = numberOf(entry, 'cost');
    const reply = messages.findLast((message) => message.role === 'architect');
    if (cost !== undefined && cost !== 0 && reply !== undefined) {
        reply.cost = cost;
        reply.duration = numberOf(entry, 'durationMs');
    }
}

/** The architect's past sessions as chat messages: a divider between sessions, each text and tool call, and a result's cost on the reply it priced. */
export function historyMessages(history: unknown): ChatMessage[] {
    const messages: ChatMessage[] = [];
    for (const [index, session] of listOf(history, 'sessions').entries()) {
        if (index > 0) {
            messages.push(sessionDivider(session));
        }
        for (const entry of listOf(session, 'messages')) {
            if (textOf(entry, 'type') === 'result') {
                priceLastReply(messages, entry);
            } else {
                const message = messageOf(entry, `hist-${index}-${messages.length}`);
                if (message !== undefined) {
                    messages.push(message);
                }
            }
        }
    }
    return messages;
}

/** The reply with new blocks appended, its content the text they carry. */
export function withBlocks(reply: ChatMessage, blocks: ActivityBlock[]): ChatMessage {
    const all = [...reply.blocks, ...blocks];
    const content = all.flatMap((block) => (block.type === 'text' ? [block.content] : [])).join('');
    return { ...reply, blocks: all, content };
}
