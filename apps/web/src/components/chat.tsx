'use client';

import { useEffect, useRef, useState } from 'react';
import type { ReactNode } from 'react';

import { Bubble, Typing } from './chat/bubble';
import type { ChatBubble } from './chat/bubble';
import { Composer } from './chat/composer';

export type { ChatBubble } from './chat/bubble';

interface ChatProps {
    messages: ChatBubble[];
    onSend: (text: string) => void;
    placeholder?: string;
    disabled?: boolean;
    emptyState?: ReactNode;
    typingText?: string;
    extras?: (message: ChatBubble, index: number) => ReactNode;
    assistantLabel?: string;
    userLabel?: string;
    focusInput?: boolean;
    className?: string;
    input?: string;
    onInputChange?: (value: string) => void;
}

interface MessagesProps {
    messages: ChatBubble[];
    extras: ChatProps['extras'];
    assistantLabel: string;
    userLabel: string;
}

function Messages({ messages, extras, assistantLabel, userLabel }: MessagesProps) {
    return (
        <>
            {messages.map((message, index) => (
                <Bubble
                    extra={extras?.(message, index)}
                    // oxlint-disable-next-line react/no-array-index-key -- a conversation only grows at its end and its messages carry no id; the index is each one's place in it
                    key={`${message.timestamp.getTime()}-${message.role}-${index}`}
                    label={message.role === 'assistant' ? assistantLabel : userLabel}
                    message={message}
                />
            ))}
        </>
    );
}

interface ComposerState {
    controlledInput: string | undefined;
    onInputChange: ((value: string) => void) | undefined;
    focusInput: boolean;
    disabled: boolean;
    onSend: (text: string) => void;
}

function useComposer({
    controlledInput,
    onInputChange,
    focusInput,
    disabled,
    onSend,
}: ComposerState) {
    const [ownInput, setOwnInput] = useState('');
    const controlled = controlledInput !== undefined && onInputChange !== undefined;
    const input = controlled ? controlledInput : ownInput;
    const setInput = controlled ? onInputChange : setOwnInput;
    const inputRef = useRef<HTMLInputElement>(null);

    useEffect(() => {
        if (focusInput) {
            inputRef.current?.focus();
        }
    }, [focusInput]);

    const send = () => {
        const text = input.trim();
        if (text === '' || disabled) {
            return;
        }
        onSend(text);
        setInput('');
        requestAnimationFrame(() => {
            inputRef.current?.focus();
        });
    };
    return { input, inputRef, send, setInput };
}

function scrollIntoView(element: HTMLDivElement | null): void {
    element?.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
}

/** Chat is the shared, presentational surface used by both the agent chat tab and the Architect chat widget. It's container-less by design - only the message bubbles and the focused input border carry backgrounds, so the chat blends into whatever page hosts it. */
export function Chat({
    messages,
    onSend,
    placeholder = 'Message...',
    disabled = false,
    emptyState,
    typingText = '',
    extras,
    assistantLabel = 'assistant',
    userLabel = 'you',
    focusInput = false,
    className = '',
    input: controlledInput,
    onInputChange,
}: ChatProps) {
    const { input, inputRef, send, setInput } = useComposer({
        controlledInput,
        disabled,
        focusInput,
        onInputChange,
        onSend,
    });

    return (
        <div className={`flex min-h-0 flex-1 flex-col ${className}`}>
            <div className="min-h-0 flex-1 space-y-3 overflow-x-hidden overflow-y-auto py-2 pr-1">
                {messages.length === 0 && emptyState !== undefined && (
                    <div className="flex h-full items-center justify-center">{emptyState}</div>
                )}
                <Messages
                    assistantLabel={assistantLabel}
                    extras={extras}
                    messages={messages}
                    userLabel={userLabel}
                />
                {disabled && typingText !== '' && <Typing text={typingText} />}
                <div key={`${messages.length}-${String(disabled)}`} ref={scrollIntoView} />
            </div>
            <Composer
                disabled={disabled}
                input={input}
                inputRef={inputRef}
                onInput={setInput}
                onSend={send}
                placeholder={placeholder}
            />
        </div>
    );
}

/** Suggestion pills shown inside the empty state. Each click fills the input - onPick is usually just setInput. Kept here so both chats can use the same styling. */
export function ChatSuggestions({
    suggestions,
    onPick,
}: {
    suggestions: string[];
    onPick: (text: string) => void;
}) {
    return (
        <div className="flex max-w-sm flex-wrap justify-center gap-2">
            {suggestions.map((suggestion) => (
                <button
                    className="text-muted-foreground/40 hover:text-foreground/70 rounded-full border border-white/[0.06] bg-white/[0.03] px-3 py-1.5 text-[11px] transition-colors hover:bg-white/[0.06]"
                    key={suggestion}
                    onClick={() => {
                        onPick(suggestion);
                    }}
                    type="button"
                >
                    {suggestion}
                </button>
            ))}
        </div>
    );
}
