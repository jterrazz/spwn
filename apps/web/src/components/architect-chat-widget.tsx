'use client';

import {
    IconArrowUp,
    IconChevronDown,
    IconHexagonFilled,
    IconMaximize,
    IconMessageCircle,
} from '@tabler/icons-react';
import { usePathname, useRouter } from 'next/navigation';
import { useEffect, useRef, useState } from 'react';

import { Chat, type ChatBubble } from '@/components/chat';
import { useArchitectChat } from '@/contexts/architect-chat-context';

function ArchitectGlyph({ isRunning, isActive }: { isRunning: boolean; isActive: boolean }) {
    return (
        <span className="relative inline-flex h-[18px] w-[18px] shrink-0 items-center justify-center self-center overflow-visible">
            <style jsx>{`
                @keyframes architect-rainbow-hue {
                    0% {
                        filter: hue-rotate(0deg) brightness(1.2);
                    }
                    100% {
                        filter: hue-rotate(360deg) brightness(1.2);
                    }
                }
            `}</style>
            {isRunning && isActive ? (
                <>
                    <IconHexagonFilled
                        className="absolute inset-0 m-auto blur-[6px] opacity-70 text-pink-400"
                        size={14}
                        style={{ animation: 'architect-rainbow-hue 3s linear infinite' }}
                    />
                    <IconHexagonFilled
                        className="absolute inset-0 m-auto text-pink-400"
                        size={14}
                        style={{ animation: 'architect-rainbow-hue 3s linear infinite' }}
                    />
                </>
            ) : (
                <span className="block leading-none translate-y-[0.5px]">
                    <IconHexagonFilled className="text-muted-foreground/45" size={14} />
                </span>
            )}
        </span>
    );
}

export function ArchitectChatWidget() {
    const pathname = usePathname();
    const router = useRouter();
    const [expanded, setExpanded] = useState(false);
    const panelRef = useRef<HTMLDivElement>(null);

    const { messages, chatInput, setChatInput, sending, sendMessage, isRunning } =
        useArchitectChat();

    // Close when clicking outside the expanded panel
    useEffect(() => {
        if (!expanded) {
            return;
        }

        const handlePointerDown = (event: MouseEvent | TouchEvent) => {
            const target = event.target as Node | null;
            if (!target) {
                return;
            }
            if (panelRef.current?.contains(target)) {
                return;
            }
            setExpanded(false);
        };

        document.addEventListener('mousedown', handlePointerDown);
        document.addEventListener('touchstart', handlePointerDown, { passive: true });
        return () => {
            document.removeEventListener('mousedown', handlePointerDown);
            document.removeEventListener('touchstart', handlePointerDown);
        };
    }, [expanded]);

    // Hide on architect page - the full page takes over
    if (pathname === '/architect' || pathname.startsWith('/architect/')) {
        return null;
    }

    // Adapt the architect context's messages into the shared ChatBubble shape.
    const bubbles: ChatBubble[] = messages.map((m) => ({
        role: m.role === 'architect' ? 'assistant' : 'user',
        blocks: m.blocks,
        content: m.content,
        timestamp: m.timestamp,
        error: m.error,
        cost: m.cost,
        duration: m.duration,
    }));

    // Chat is controlled via chatInput/setChatInput - at send time the
    // Context's chatInput is already the latest value, so sendMessage()
    // Reads it directly. The `text` arg is the same as chatInput here.
    const handleSend = () => {
        void sendMessage();
    };

    // ── Expanded panel ──
    if (expanded) {
        return (
            <div
                className="fixed bottom-4 right-4 z-[200] w-[420px] h-[540px] flex flex-col rounded-2xl border border-white/[0.08] bg-background/95 backdrop-blur-xl shadow-2xl shadow-black/30 animate-in slide-in-from-bottom-4 fade-in duration-200 overflow-hidden"
                ref={panelRef}
            >
                {/* Header */}
                <div className="flex items-center justify-between px-4 py-3 border-b border-white/[0.06]">
                    <div className="flex items-center gap-1 rounded-full border border-foreground/[0.08] dark:border-white/[0.1] bg-foreground/[0.04] dark:bg-white/[0.05] px-2.5 py-1.5 shadow-[inset_0_1px_0_rgba(255,255,255,0.08),0_1px_2px_rgba(0,0,0,0.04)] dark:shadow-[inset_0_1px_0_rgba(255,255,255,0.05),0_1px_2px_rgba(0,0,0,0.18)] backdrop-blur-md">
                        <ArchitectGlyph isActive={sending} isRunning={isRunning} />
                    </div>
                    <div className="flex items-center gap-1">
                        <button
                            className="w-7 h-7 flex items-center justify-center rounded-md text-muted-foreground/30 hover:text-foreground/60 transition-colors"
                            onClick={() => {
                                setExpanded(false);
                                router.push('/architect');
                            }}
                            title="Open full page"
                        >
                            <IconMaximize size={14} />
                        </button>
                        <button
                            className="w-7 h-7 flex items-center justify-center rounded-md text-muted-foreground/30 hover:text-foreground/60 transition-colors"
                            onClick={() => setExpanded(false)}
                            title="Minimize"
                        >
                            <IconChevronDown size={14} />
                        </button>
                    </div>
                </div>

                {/* Messages + input via shared Chat - indexed by the original
            messages array so the extras closure can read stackAction etc. */}
                <Chat
                    autoFocus={expanded}
                    className="flex-1 px-3 pb-3"
                    disabled={sending}
                    emptyState={
                        <div className="flex flex-col items-center justify-center text-center">
                            <IconMessageCircle
                                className="text-muted-foreground/15 mb-2"
                                size={24}
                            />
                            <p className="text-xs text-muted-foreground/30">
                                Talk to the Architect
                            </p>
                            <p className="text-[10px] text-muted-foreground/20 mt-1 max-w-[260px]">
                                Ask anything - create agents, manage worlds, or check status.
                            </p>
                            {!isRunning && (
                                <p className="text-[9px] text-yellow-400/40 mt-2 font-mono">
                                    Architect is offline - start it from the Architect page
                                </p>
                            )}
                        </div>
                    }
                    extras={() => null}
                    input={chatInput}
                    messages={bubbles}
                    onInputChange={setChatInput}
                    onSend={handleSend}
                    placeholder="Talk to the Architect..."
                    typingText="Thinking…"
                />
            </div>
        );
    }

    // ── Collapsed bar ──

    // When offline: show a compact button to navigate to architect page
    if (!isRunning) {
        return (
            <div className="fixed bottom-4 right-4 z-[200] animate-in fade-in duration-200">
                <button
                    className="flex items-center gap-2.5 rounded-full border border-foreground/[0.08] dark:border-white/[0.1] bg-foreground/[0.04] dark:bg-white/[0.05] backdrop-blur-md shadow-[inset_0_1px_0_rgba(255,255,255,0.08),0_1px_2px_rgba(0,0,0,0.04)] dark:shadow-[inset_0_1px_0_rgba(255,255,255,0.05),0_1px_2px_rgba(0,0,0,0.18)] px-4 py-2.5 hover:bg-white/[0.08] transition-colors"
                    onClick={() => router.push('/architect')}
                    title="Architect is offline - click to start"
                >
                    <ArchitectGlyph isActive={false} isRunning={false} />
                    <span className="text-[12px] text-muted-foreground/35">Architect offline</span>
                </button>
            </div>
        );
    }

    return (
        <>
            <div className="fixed bottom-4 right-4 z-[200] animate-in fade-in duration-200">
                <div className="flex items-center gap-2 rounded-full border border-foreground/[0.08] dark:border-white/[0.1] bg-foreground/[0.04] dark:bg-white/[0.05] backdrop-blur-md shadow-[inset_0_1px_0_rgba(255,255,255,0.08),0_1px_2px_rgba(0,0,0,0.04)] dark:shadow-[inset_0_1px_0_rgba(255,255,255,0.05),0_1px_2px_rgba(0,0,0,0.18)] px-2.5 py-1.5 w-[300px]">
                    <button
                        className="flex h-[30px] items-center justify-center rounded-full border border-transparent px-2 shrink-0"
                        onClick={() => setExpanded(true)}
                        title="Architect alive"
                    >
                        <ArchitectGlyph isActive={sending} isRunning={isRunning} />
                    </button>
                    <input
                        className="flex-1 bg-transparent text-[13px] text-foreground/80 placeholder:text-muted-foreground/25 focus:outline-none min-w-0"
                        disabled={sending}
                        onChange={(e) => setChatInput(e.target.value)}
                        onFocus={() => setExpanded(true)}
                        onKeyDown={(e) => {
                            if (e.key === 'Enter' && !e.shiftKey) {
                                e.preventDefault();
                                if (chatInput.trim()) {
                                    handleSend();
                                    setExpanded(true);
                                }
                            }
                        }}
                        placeholder="Ask the Architect..."
                        value={chatInput}
                    />
                    {chatInput.trim() ? (
                        <button
                            className="p-1.5 rounded-full bg-white/[0.08] text-foreground/70 hover:bg-white/[0.12] transition-all shrink-0"
                            onClick={() => {
                                handleSend();
                                setExpanded(true);
                            }}
                        >
                            <IconArrowUp size={14} />
                        </button>
                    ) : null}
                </div>
            </div>
        </>
    );
}
