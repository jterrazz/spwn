'use client';

import type { ReactNode } from 'react';

import type { ActivityBlock } from '../../activity/blocks';
import { ActivityMessageView } from '../activity-blocks';

/** ChatBubble is the minimal shape any caller must provide. The chat does not manage state - the parent owns the message list and passes it in. */
export interface ChatBubble {
    role: 'assistant' | 'user';
    blocks: ActivityBlock[];
    content: string;
    timestamp: Date;
    error?: boolean | undefined;
    cost?: number | undefined;
    duration?: number | undefined;
}

function toneOf(message: ChatBubble): string {
    if (message.role === 'user') {
        return 'bg-white/[0.08] text-foreground/85';
    }
    return message.error === true
        ? 'bg-red-500/10 border border-red-500/15 text-red-400/80'
        : 'bg-white/[0.03] border border-white/[0.06] text-foreground/75';
}

function textOf(message: ChatBubble): string {
    if (message.content !== '') {
        return message.content;
    }
    return message.role === 'assistant' ? '...' : '';
}

interface BubbleProps {
    message: ChatBubble;
    label: string;
    extra: ReactNode;
}

export function Bubble({ message, label, extra }: BubbleProps) {
    return (
        <div className={`flex flex-col ${message.role === 'user' ? 'items-end' : 'items-start'}`}>
            <div
                className={`max-w-[85%] rounded-2xl px-3.5 py-2.5 transition-colors ${toneOf(message)}`}
            >
                {message.role === 'assistant' && message.blocks.length > 0 ? (
                    <ActivityMessageView
                        message={{
                            blocks: message.blocks,
                            cost: message.cost,
                            duration: message.duration,
                            role: 'agent',
                            timestamp: message.timestamp,
                        }}
                    />
                ) : (
                    <p className="text-xs leading-relaxed break-words whitespace-pre-wrap">
                        {textOf(message)}
                    </p>
                )}
                <p className="text-muted-foreground/25 mt-1 text-[9px]">
                    {label}
                    {' · '}
                    {message.timestamp.toLocaleTimeString([], {
                        hour: '2-digit',
                        minute: '2-digit',
                    })}
                </p>
            </div>
            {extra}
        </div>
    );
}

export function Typing({ text }: { text: string }) {
    return (
        <div className="flex items-start">
            <div className="rounded-2xl border border-white/[0.06] bg-white/[0.03] px-3.5 py-2.5">
                <div className="flex items-center gap-2">
                    <div className="flex gap-1">
                        {['0ms', '150ms', '300ms'].map((delay) => (
                            <span
                                className="bg-foreground/30 h-1.5 w-1.5 animate-bounce rounded-full"
                                key={delay}
                                style={{ animationDelay: delay }}
                            />
                        ))}
                    </div>
                    <span className="text-muted-foreground/40 text-xs">{text}</span>
                </div>
            </div>
        </div>
    );
}
