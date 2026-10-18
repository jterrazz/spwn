'use client';

import { IconChevronDown, IconMaximize, IconMessageCircle } from '@tabler/icons-react';

import type { ChatMessage } from '../../domain/architect';
import { Chat } from '../chat';
import type { ChatBubble } from '../chat';
import { ArchitectGlyph } from './glyph';

const PILL =
    'border-foreground/[0.08] bg-foreground/[0.04] shadow-[inset_0_1px_0_rgba(255,255,255,0.08),0_1px_2px_rgba(0,0,0,0.04)] backdrop-blur-md dark:border-white/[0.1] dark:bg-white/[0.05] dark:shadow-[inset_0_1px_0_rgba(255,255,255,0.05),0_1px_2px_rgba(0,0,0,0.18)]';
const ICON_BUTTON =
    'text-muted-foreground/30 hover:text-foreground/60 flex h-7 w-7 items-center justify-center rounded-md transition-colors';

function bubbleOf(message: ChatMessage): ChatBubble {
    return { ...message, role: message.role === 'architect' ? 'assistant' : 'user' };
}

function Empty({ running }: { running: boolean }) {
    return (
        <div className="flex flex-col items-center justify-center text-center">
            <IconMessageCircle className="text-muted-foreground/15 mb-2" size={24} />
            <p className="text-muted-foreground/30 text-xs">Talk to the Architect</p>
            <p className="text-muted-foreground/20 mt-1 max-w-[260px] text-[10px]">
                Ask anything - create agents, manage worlds, or check status.
            </p>
            {!running && (
                <p className="mt-2 font-mono text-[9px] text-yellow-400/40">
                    Architect is offline - start it from the Architect page
                </p>
            )}
        </div>
    );
}

interface ArchitectPanelProps {
    panelRef: React.RefObject<HTMLDivElement | null>;
    messages: ChatMessage[];
    input: string;
    sending: boolean;
    running: boolean;
    onInput: (value: string) => void;
    onSend: () => void;
    onOpenPage: () => void;
    onMinimize: () => void;
}

/** The architect's chat, unfolded in the corner of any page. */
export function ArchitectPanel({
    panelRef,
    messages,
    input,
    sending,
    running,
    onInput,
    onSend,
    onOpenPage,
    onMinimize,
}: ArchitectPanelProps) {
    return (
        <div
            className="bg-background/95 animate-in slide-in-from-bottom-4 fade-in fixed right-4 bottom-4 z-[200] flex h-[540px] w-[420px] flex-col overflow-hidden rounded-2xl border border-white/[0.08] shadow-2xl shadow-black/30 backdrop-blur-xl duration-200"
            ref={panelRef}
        >
            <div className="flex items-center justify-between border-b border-white/[0.06] px-4 py-3">
                <div
                    className={`flex items-center gap-1 rounded-full border px-2.5 py-1.5 ${PILL}`}
                >
                    <ArchitectGlyph isActive={sending} isRunning={running} />
                </div>
                <div className="flex items-center gap-1">
                    <button
                        aria-label="Open full page"
                        className={ICON_BUTTON}
                        onClick={onOpenPage}
                        title="Open full page"
                        type="button"
                    >
                        <IconMaximize size={14} />
                    </button>
                    <button
                        aria-label="Minimize"
                        className={ICON_BUTTON}
                        onClick={onMinimize}
                        title="Minimize"
                        type="button"
                    >
                        <IconChevronDown size={14} />
                    </button>
                </div>
            </div>
            <Chat
                className="flex-1 px-3 pb-3"
                disabled={sending}
                emptyState={<Empty running={running} />}
                focusInput
                input={input}
                messages={messages.map(bubbleOf)}
                onInputChange={onInput}
                onSend={onSend}
                placeholder="Talk to the Architect..."
                typingText="Thinking..."
            />
        </div>
    );
}

export { PILL };
