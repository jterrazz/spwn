'use client';

import { useCallback } from 'react';

import { withBlocks } from '../activity/blocks';
import type { ActivityBlock } from '../activity/blocks';
import { streamChat } from '../api/stream-chat';

export interface Reply {
    blocks: ActivityBlock[];
    content: string;
    error?: boolean | undefined;
    cost?: number | undefined;
    duration?: number | undefined;
}

/** Appends an empty reply to a conversation, then fills it from a streamed chat answer: its blocks as they arrive, its cost when it ends, its error if it fails. */
export function useReplyStream<Message extends Reply>(
    setMessages: React.Dispatch<React.SetStateAction<Message[]>>,
    isReply: (message: Message) => boolean,
) {
    return useCallback(
        async (url: string, body: Record<string, unknown>, empty: Message) => {
            const update = (change: (reply: Message) => Message) => {
                setMessages((previous) => {
                    const last = previous.at(-1);
                    return last !== undefined && isReply(last)
                        ? [...previous.slice(0, -1), change(last)]
                        : previous;
                });
            };
            setMessages((previous) => [...previous, empty]);
            await streamChat({
                body,
                onBlocks: (blocks) => {
                    update((reply) => withBlocks(reply, blocks));
                },
                onDone: (meta) => {
                    update((reply) => ({ ...reply, cost: meta.cost, duration: meta.duration }));
                },
                onError: (error) => {
                    update((reply) => ({
                        ...withBlocks(reply, [{ content: error, type: 'error' }]),
                        content: error,
                        error: true,
                    }));
                },
                url,
            });
        },
        [setMessages, isReply],
    );
}
