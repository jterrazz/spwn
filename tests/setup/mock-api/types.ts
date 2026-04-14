export interface MockApiCall {
    timestamp: Date;
    model: string;
    messages: Message[];
    tools: string[];
}

export interface Message {
    role: 'assistant' | 'user';
    content: ContentBlock[] | string;
}

export interface ContentBlock {
    type: 'text' | 'tool_result' | 'tool_use';
    text?: string;
    id?: string;
    name?: string;
    input?: Record<string, unknown>;
    tool_use_id?: string;
    content?: string;
}

export interface ScriptedResponse {
    text?: string;
    toolCalls?: ToolCall[];
    stopReason?: 'end_turn' | 'tool_use';
}

export interface ToolCall {
    name: string;
    input: Record<string, unknown>;
}

export type ConversationScript = (messages: Message[]) => ScriptedResponse;
