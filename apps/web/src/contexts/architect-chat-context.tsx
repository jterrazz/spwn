'use client';

import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';

import type { ActivityBlock } from '@/lib/activity-types';
import { apiGet, goApiUrl } from '@/lib/api-client';
import { streamChat } from '@/lib/stream-chat';

// ── Types ──

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
    error?: boolean;
    cost?: number;
    duration?: number;
}

// ── Context ──

interface ArchitectChatContextValue {
    messages: ChatMessage[];
    chatInput: string;
    setChatInput: (value: string) => void;
    sending: boolean;
    sendMessage: () => Promise<void>;
    architectStatus: ArchitectStatus | null;
    isRunning: boolean;
    highlightTitle: null | string;
    refreshStatus: () => void;
    setArchitectStatus: React.Dispatch<React.SetStateAction<ArchitectStatus | null>>;
    loading: boolean;
}

const ArchitectChatContext = createContext<ArchitectChatContextValue | null>(null);

export function useArchitectChat() {
    const ctx = useContext(ArchitectChatContext);
    if (!ctx) {
        throw new Error('useArchitectChat must be used within ArchitectChatProvider');
    }
    return ctx;
}

export function ArchitectChatProvider({ children }: { children: React.ReactNode }) {
    const [messages, setMessages] = useState<ChatMessage[]>([]);
    const [chatInput, setChatInput] = useState('');
    const [sending, setSending] = useState(false);
    const [architectStatus, setArchitectStatus] = useState<ArchitectStatus | null>(null);
    const [loading, setLoading] = useState(true);

    const refreshStatus = useCallback(() => {
        apiGet<ArchitectStatus>('/api/architect/status')
            .catch(() => ({ status: 'stopped' as const, containerId: null, uptime: null }))
            .then((archStatus) => {
                setArchitectStatus(archStatus);
                setLoading(false);
            });
    }, []);

    // Load conversation history on mount
    useEffect(() => {
        apiGet<{
            sessions: {
                id: string;
                messages: {
                    role: string;
                    content: string;
                    timestamp: string;
                    type: string;
                    toolName?: string;
                    cost?: number;
                    durationMs?: number;
                }[];
                startedAt: string;
                cost?: number;
            }[];
        }>('/api/architect/history')
            .then((data) => {
                if (!data.sessions || data.sessions.length === 0) {
                    return;
                }
                const historyMsgs: ChatMessage[] = [];
                for (let si = 0; si < data.sessions.length; si++) {
                    const session = data.sessions[si];
                    if (si > 0) {
                        historyMsgs.push({
                            role: 'architect',
                            content: `── Session ${session.id.slice(0, 8)} · ${session.startedAt ? new Date(session.startedAt).toLocaleString() : 'unknown'} ──`,
                            blocks: [{ type: 'text' as const, content: `── Previous session ──` }],
                            timestamp: session.startedAt ? new Date(session.startedAt) : new Date(),
                        });
                    }
                    for (const msg of session.messages) {
                        if (msg.type === 'text' && msg.role === 'user') {
                            historyMsgs.push({
                                role: 'user',
                                content: msg.content,
                                blocks: [{ type: 'text' as const, content: msg.content }],
                                timestamp: msg.timestamp ? new Date(msg.timestamp) : new Date(),
                            });
                        } else if (msg.type === 'text' && msg.role === 'assistant') {
                            historyMsgs.push({
                                role: 'architect',
                                content: msg.content,
                                blocks: [{ type: 'text' as const, content: msg.content }],
                                timestamp: msg.timestamp ? new Date(msg.timestamp) : new Date(),
                            });
                        } else if (msg.type === 'tool_use') {
                            historyMsgs.push({
                                role: 'architect',
                                content: `🔧 ${msg.toolName || 'tool'}`,
                                blocks: [
                                    {
                                        type: 'tool_use',
                                        tool: msg.toolName || 'tool',
                                        input: {},
                                        id: `hist-${si}-${historyMsgs.length}`,
                                    } as ActivityBlock,
                                ],
                                timestamp: msg.timestamp ? new Date(msg.timestamp) : new Date(),
                            });
                        } else if (msg.type === 'result' && msg.cost) {
                            const lastArch = [...historyMsgs]
                                .toReversed()
                                .find((m) => m.role === 'architect');
                            if (lastArch) {
                                lastArch.cost = msg.cost;
                                lastArch.duration = msg.durationMs;
                            }
                        }
                    }
                }
                if (historyMsgs.length > 0) {
                    setMessages(historyMsgs);
                }
            })
            .catch(() => {});
    }, []);

    // Status polling
    useEffect(() => {
        refreshStatus();
        const interval = setInterval(refreshStatus, 10_000);
        return () => clearInterval(interval);
    }, [refreshStatus]);

    const doTalk = useCallback(async (msg: string) => {
        let msgIndex: number;
        setMessages((prev) => {
            msgIndex = prev.length;
            return [
                ...prev,
                {
                    role: 'architect' as const,
                    content: '',
                    blocks: [],
                    timestamp: new Date(),
                },
            ];
        });

        await streamChat({
            url: goApiUrl('/api/architect/talk'),
            body: { message: msg },
            onBlocks: (newBlocks) => {
                setMessages((prev) => {
                    const updated = [...prev];
                    const last = updated[msgIndex!];
                    if (last && last.role === 'architect') {
                        const allBlocks = [...last.blocks, ...newBlocks];
                        const textContent = allBlocks
                            .filter((b) => b.type === 'text')
                            .map((b) => (b as { content: string }).content)
                            .join('');
                        updated[msgIndex!] = { ...last, blocks: allBlocks, content: textContent };
                    }
                    return updated;
                });
            },
            onDone: (meta) => {
                setMessages((prev) => {
                    const updated = [...prev];
                    const last = updated[msgIndex!];
                    if (last && last.role === 'architect') {
                        updated[msgIndex!] = { ...last, cost: meta.cost, duration: meta.duration };
                    }
                    return updated;
                });
            },
            onError: (error) => {
                setMessages((prev) => {
                    const updated = [...prev];
                    const last = updated[msgIndex!];
                    if (last && last.role === 'architect') {
                        updated[msgIndex!] = {
                            ...last,
                            blocks: [...last.blocks, { type: 'error' as const, content: error }],
                            content: error,
                            error: true,
                        };
                    }
                    return updated;
                });
            },
        });
    }, []);

    const sendMessage = useCallback(async () => {
        const msg = chatInput.trim();
        if (!msg || sending) {
            return;
        }

        const userMsg: ChatMessage = {
            role: 'user',
            content: msg,
            blocks: [{ type: 'text', content: msg }],
            timestamp: new Date(),
        };
        setMessages((prev) => [...prev, userMsg]);
        setChatInput('');
        setSending(true);

        try {
            const running = architectStatus?.status === 'running';

            if (!running) {
                setMessages((prev) => [
                    ...prev,
                    {
                        role: 'architect',
                        content: 'Architect is offline. Start it first using the button above.',
                        blocks: [
                            {
                                type: 'text' as const,
                                content:
                                    'Architect is offline. Start it first using the button above.',
                            },
                        ],
                        timestamp: new Date(),
                        error: true,
                    },
                ]);
                setSending(false);
                return;
            }

            await doTalk(msg);
        } catch (error: unknown) {
            const errMsg = error instanceof Error ? error.message : 'Unknown error';
            setMessages((prev) => [
                ...prev,
                {
                    role: 'architect',
                    content: `Error: ${errMsg}`,
                    blocks: [{ type: 'error' as const, content: errMsg }],
                    timestamp: new Date(),
                    error: true,
                },
            ]);
        } finally {
            setSending(false);
        }
    }, [chatInput, sending, architectStatus, doTalk]);

    const isRunning = architectStatus?.status === 'running';

    const contextValue = useMemo(
        () => ({
            messages,
            chatInput,
            setChatInput,
            sending,
            sendMessage,
            architectStatus,
            isRunning,
            highlightTitle: null,
            refreshStatus,
            setArchitectStatus,
            loading,
        }),
        [
            messages,
            chatInput,
            sending,
            sendMessage,
            architectStatus,
            isRunning,
            refreshStatus,
            loading,
        ],
    );

    return (
        <ArchitectChatContext.Provider value={contextValue}>
            {children}
        </ArchitectChatContext.Provider>
    );
}
