'use client';

import { createContext, useCallback, useContext, useMemo, useState } from 'react';

import { apiGet, goApiUrl } from '../api/client';
import { failed, historyMessages, said, STOPPED_ARCHITECT } from '../domain/architect';
import type { ArchitectStatus, ChatMessage } from '../domain/architect';
import { useBackgroundTask } from '../hooks/use-background-task';
import { useReplyStream } from '../hooks/use-reply-stream';

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

function isReply(message: ChatMessage): boolean {
    return message.role === 'architect';
}

function useConversation(running: boolean) {
    const [messages, setMessages] = useState<ChatMessage[]>([]);
    const [chatInput, setChatInput] = useState('');
    const [sending, setSending] = useState(false);
    const stream = useReplyStream(setMessages, isReply);

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
            await stream(
                goApiUrl('/api/architect/talk'),
                { message },
                { blocks: [], content: '', role: 'architect', timestamp: new Date() },
            );
        } catch (error: unknown) {
            const reason = error instanceof Error ? error.message : 'Unknown error';
            setMessages((previous) => [...previous, failed(reason, `Error: ${reason}`)]);
        } finally {
            setSending(false);
        }
    }, [chatInput, sending, running, stream]);

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
