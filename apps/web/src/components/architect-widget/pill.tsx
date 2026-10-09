'use client';

import { IconArrowUp } from '@tabler/icons-react';

import { ArchitectGlyph } from './glyph';
import { PILL } from './panel';

interface ArchitectPillProps {
    input: string;
    sending: boolean;
    onInput: (value: string) => void;
    onSend: () => void;
    onExpand: () => void;
}

/** The architect folded into a one-line prompt; focusing it unfolds the chat, sending opens it on the reply. */
export function ArchitectPill({ input, sending, onInput, onSend, onExpand }: ArchitectPillProps) {
    const send = () => {
        if (input.trim() !== '') {
            onSend();
            onExpand();
        }
    };
    return (
        <div className="animate-in fade-in fixed right-4 bottom-4 z-[200] duration-200">
            <div
                className={`flex w-[300px] items-center gap-2 rounded-full border px-2.5 py-1.5 ${PILL}`}
            >
                <button
                    aria-label="Architect alive"
                    className="flex h-[30px] shrink-0 items-center justify-center rounded-full border border-transparent px-2"
                    onClick={onExpand}
                    title="Architect alive"
                    type="button"
                >
                    <ArchitectGlyph isActive={sending} isRunning />
                </button>
                <input
                    aria-label="Ask the Architect"
                    className="text-foreground/80 placeholder:text-muted-foreground/25 min-w-0 flex-1 bg-transparent text-[13px] focus:outline-none"
                    disabled={sending}
                    onChange={(event) => {
                        onInput(event.target.value);
                    }}
                    onFocus={onExpand}
                    onKeyDown={(event) => {
                        if (event.key === 'Enter' && !event.shiftKey) {
                            event.preventDefault();
                            send();
                        }
                    }}
                    placeholder="Ask the Architect..."
                    value={input}
                />
                {input.trim() !== '' && (
                    <button
                        aria-label="Send"
                        className="text-foreground/70 shrink-0 rounded-full bg-white/[0.08] p-1.5 transition-all hover:bg-white/[0.12]"
                        onClick={send}
                        type="button"
                    >
                        <IconArrowUp size={14} />
                    </button>
                )}
            </div>
        </div>
    );
}
