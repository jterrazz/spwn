'use client';

import { createContext, useCallback, useContext, useMemo, useState } from 'react';

import type { ActivityBlock } from '../activity/blocks';
import { apiGet, goApiUrl } from '../api/client';
import { streamChat } from '../api/stream-chat';
import { failed, historyMessages, said, STOPPED_ARCHITECT, withBlocks } from '../domain/architect';
import type { ArchitectStatus, ChatMessage } from '../domain/architect';
import { useBackgroundTask } from '../hooks/use-background-task';

interface ArchitectChatContextValue {
    messages: ChatMessage[];
    chatInput: string;
    setChatInput: (value: string) => void;
    sending: boolean;
    sendMessage: () => Promise<void>;
    architectStatus: ArchitectStatus | null;
    isRunning: boolean;
    setArchitectStatus: React.Dispatch<React.SetStateAction<ArchitectStatus | null>>;
    loading: boolean;
}

const ArchitectChatContext = createContext<ArchitectChatContextValue | null>(null);

const OFFLINE = 'Architect is offline. Start it first using the button above.';

export function useArchitectChat() {
    const context = useContext(ArchitectChatContext);
    if (context === null) {
        throw new Error('useArchitectChat must be used within ArchitectChatProvider');
    }
    return context;
}

function useArchitectStatus() {
    const [architectStatus, setArchitectStatus] = useState<ArchitectStatus | null>(null);
    const [loading, setLoading] = useState(true);

    const refreshStatus = useCallback(async () => {
        try {
            setArchitectStatus(await apiGet<ArchitectStatus>('/api/architect/status'));
        } catch {
            setArchitectStatus(STOPPED_ARCHITECT);
        }
        setLoading(false);
    }, []);

    useBackgroundTask(refreshStatus, 10_000);
    return { architectStatus, loading, setArchitectStatus };
}

function useTalk(setMessages: React.Dispatch<React.SetStateAction<ChatMessage[]>>) {
    return useCallback(
        async (message: string) => {
            const reply = (update: (last: ChatMessage) => ChatMessage) => {
                setMessages((previous) => {
                    const last = previous.at(-1);
                    return last?.role === 'architect'
                        ? [...previous.slice(0, -1), update(last)]
                        : previous;
                });
            };
            const failure = (error: string) => (last: ChatMessage) => {
                const block: ActivityBlock = { content: error, type: 'error' };
                return { ...withBlocks(last, [block]), content: error, error: true };
            };
            setMessages((previous) => [
                ...previous,
                { blocks: [], content: '', role: 'architect', timestamp: new Date() },
            ]);

            await streamChat({
                body: { message },
                onBlocks: (blocks) => {
                    reply((last) => withBlocks(last, blocks));
                },
                onDone: (meta) => {
                    reply((last) => ({ ...last, cost: meta.cost, duration: meta.duration }));
                },
                onError: (error) => {
                    reply(failure(error));
                },
                url: goApiUrl('/api/architect/talk'),
            });
        },
        [setMessages],
    );
}

function useConversation(running: boolean) {
    const [messages, setMessages] = useState<ChatMessage[]>([]);
    const [chatInput, setChatInput] = useState('');
    const [sending, setSending] = useState(false);
    const talk = useTalk(setMessages);

    const loadHistory = useCallback(async () => {
        try {
            const history = historyMessages(await apiGet<unknown>('/api/architect/history'));
            if (history.length > 0) {
                setMessages(history);
            }
        } catch {
            setMessages([]);
        }
    }, []);
    useBackgroundTask(loadHistory);

    const sendMessage = useCallback(async () => {
        const message = chatInput.trim();
        if (message === '' || sending) {
            return;
        }
        setMessages((previous) => [...previous, said('user', message)]);
        setChatInput('');
        if (!running) {
            setMessages((previous) => [
                ...previous,
                { ...said('architect', OFFLINE), error: true },
            ]);
            return;
        }
        setSending(true);
        try {
            await talk(message);
        } catch (error: unknown) {
            const reason = error instanceof Error ? error.message : 'Unknown error';
            setMessages((previous) => [...previous, failed(reason, `Error: ${reason}`)]);
        } finally {
            setSending(false);
        }
    }, [chatInput, sending, running, talk]);

    return { chatInput, messages, sending, sendMessage, setChatInput };
}

export function ArchitectChatProvider({ children }: { children: React.ReactNode }) {
    const { architectStatus, loading, setArchitectStatus } = useArchitectStatus();
    const isRunning = architectStatus?.status === 'running';
    const { chatInput, messages, sending, sendMessage, setChatInput } = useConversation(isRunning);

    const contextValue = useMemo(
        () => ({
            architectStatus,
            chatInput,
            isRunning,
            loading,
            messages,
            sending,
            sendMessage,
            setArchitectStatus,
            setChatInput,
        }),
        [
            architectStatus,
            chatInput,
            isRunning,
            loading,
            messages,
            sending,
            sendMessage,
            setArchitectStatus,
            setChatInput,
        ],
    );

    return (
        <ArchitectChatContext.Provider value={contextValue}>
            {children}
        </ArchitectChatContext.Provider>
    );
}
