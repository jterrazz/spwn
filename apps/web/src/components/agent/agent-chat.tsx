'use client';

import { IconTerminal } from '@tabler/icons-react';
import { useCallback, useState } from 'react';

import { apiGet, encPath, goApiUrl } from '../../api/client';
import { listOf, textOf } from '../../api/fields';
import { inBackground } from '../../events/in-background';
import { useBackgroundTask } from '../../hooks/use-background-task';
import { useReplyStream } from '../../hooks/use-reply-stream';
import { Chat, ChatSuggestions } from '../chat';
import type { ChatBubble } from '../chat';

const SUGGESTIONS = ['What are you working on?', 'Show me the project structure', 'Run the tests'];

function bubbleOf(entry: unknown): ChatBubble[] {
    const content = textOf(entry, 'content');
    if (textOf(entry, 'type') !== 'text' || content === '') {
        return [];
    }
    return [
        {
            blocks: [{ content, type: 'text' }],
            content,
            role: textOf(entry, 'role') === 'user' ? 'user' : 'assistant',
            timestamp: new Date(textOf(entry, 'timestamp')),
        },
    ];
}

function isReply(message: ChatBubble): boolean {
    return message.role === 'assistant';
}

function useAgentConversation(agent: string, worldId: string | undefined) {
    const inWorld = worldId !== undefined && worldId !== '';
    const [messages, setMessages] = useState<ChatBubble[]>([]);
    const [sending, setSending] = useState(false);
    const stream = useReplyStream(setMessages, isReply);

    const loadHistory = useCallback(async () => {
        if (!inWorld) {
            return;
        }
        try {
            const history = await apiGet<unknown>(
                `/api/worlds/${worldId}/history?agent=${encPath(agent)}`,
            );
            const bubbles = listOf(history, 'sessions').flatMap((session) =>
                listOf(session, 'messages').flatMap(bubbleOf),
            );
            if (bubbles.length > 0) {
                setMessages(bubbles);
            }
        } catch {
            setMessages([]);
        }
    }, [agent, inWorld, worldId]);
    useBackgroundTask(loadHistory);

    const send = async (message: string) => {
        setMessages((previous) => [
            ...previous,
            {
                blocks: [{ content: message, type: 'text' }],
                content: message,
                role: 'user',
                timestamp: new Date(),
            },
        ]);
        setSending(true);
        const url = inWorld
            ? goApiUrl(`/api/worlds/${worldId}/talk`)
            : goApiUrl(`/api/agents/${encPath(agent)}/talk`);
        const body = inWorld ? { agent, message } : { message };
        await stream(url, body, {
            blocks: [],
            content: '',
            role: 'assistant',
            timestamp: new Date(),
        });
        setSending(false);
    };

    return { messages, send, sending };
}

/** The conversation with the agent: in its world when the address names one, else with the agent alone. */
export function AgentChat({ agent, worldId }: { agent: string; worldId?: string | undefined }) {
    const { messages, send, sending } = useAgentConversation(agent, worldId);
    return (
        <Chat
            assistantLabel={agent}
            className="h-[calc(100vh-220px)] min-h-[360px]"
            disabled={sending}
            emptyState={
                <div className="flex flex-col items-center justify-center text-center">
                    <IconTerminal className="text-muted-foreground/15 mb-3" size={28} />
                    <p className="text-muted-foreground/30 text-sm">Chat with {agent}</p>
                    <p className="text-muted-foreground/20 mt-1 mb-4 text-[11px]">
                        Send messages directly to this agent in real-time
                    </p>
                    <ChatSuggestions onPick={inBackground(send)} suggestions={SUGGESTIONS} />
                </div>
            }
            focusInput
            messages={messages}
            onSend={inBackground(send)}
            placeholder={`Message ${agent}...`}
            typingText={`${agent} is thinking…`}
        />
    );
}
