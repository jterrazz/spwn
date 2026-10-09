'use client';

import { IconArrowUp } from '@tabler/icons-react';

interface ComposerProps {
    input: string;
    placeholder: string;
    disabled: boolean;
    inputRef: React.RefObject<HTMLInputElement | null>;
    onInput: (value: string) => void;
    onSend: () => void;
}

/** The chat's input line and its send button; Enter sends, Shift+Enter does not. */
export function Composer({
    input,
    placeholder,
    disabled,
    inputRef,
    onInput,
    onSend,
}: ComposerProps) {
    const ready = input.trim() !== '';
    return (
        <div className="shrink-0 pt-3">
            <div className="flex items-center gap-2 rounded-full border border-white/[0.08] bg-white/[0.04] px-3 py-2 shadow-[inset_0_1px_0_rgba(255,255,255,0.05),0_1px_2px_rgba(0,0,0,0.18)] backdrop-blur-md transition-colors focus-within:border-white/[0.14] focus-within:bg-white/[0.06]">
                <input
                    aria-label={placeholder}
                    className="text-foreground/85 placeholder:text-muted-foreground/30 min-w-0 flex-1 bg-transparent px-1 text-sm focus:outline-none disabled:cursor-not-allowed disabled:opacity-50"
                    disabled={disabled}
                    onChange={(event) => {
                        onInput(event.target.value);
                    }}
                    onKeyDown={(event) => {
                        if (event.key === 'Enter' && !event.shiftKey) {
                            event.preventDefault();
                            onSend();
                        }
                    }}
                    placeholder={placeholder}
                    ref={inputRef}
                    value={input}
                />
                <button
                    aria-label="Send"
                    className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-full transition-all disabled:cursor-not-allowed disabled:opacity-20 ${ready ? 'bg-foreground/90 text-background hover:bg-foreground' : 'text-muted-foreground/40'}`}
                    disabled={!ready || disabled}
                    onClick={onSend}
                    type="button"
                >
                    <IconArrowUp size={15} stroke={2.4} />
                </button>
            </div>
        </div>
    );
}
