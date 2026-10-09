'use client';

import { IconHexagonFilled, IconPlayerPlay, IconTerminal2 } from '@tabler/icons-react';

/** The architect page while the architect is offline: what it does, and the button that starts it. */
export function OfflineView({ onStart, disabled }: { onStart: () => void; disabled: boolean }) {
    return (
        <div className="-mt-12 flex flex-1 items-center justify-center">
            <div className="flex max-w-md flex-col items-center text-center">
                <div className="mb-6 flex h-16 w-16 items-center justify-center rounded-2xl border border-white/[0.08] bg-white/[0.04]">
                    <IconHexagonFilled className="text-muted-foreground/20" size={28} />
                </div>

                <h2 className="font-heading text-foreground/70 mb-2 text-lg tracking-wide">
                    Architect is offline
                </h2>
                <p className="text-muted-foreground/40 mb-8 text-sm leading-relaxed">
                    The Architect runs in the background and manages everything for you: creating
                    agents, spawning worlds, and keeping track of tasks. Start it to begin working.
                </p>

                <button
                    className="group flex items-center gap-3 rounded-xl border border-white/[0.10] bg-white/[0.06] px-6 py-3 transition-all duration-200 hover:border-white/[0.16] hover:bg-white/[0.10] disabled:opacity-40"
                    disabled={disabled}
                    onClick={onStart}
                    type="button"
                >
                    <IconPlayerPlay
                        className="text-green-400/80 group-hover:text-green-400"
                        size={18}
                    />
                    <span className="text-foreground/70 group-hover:text-foreground/90 text-sm font-medium">
                        Start Architect
                    </span>
                </button>

                <div className="text-muted-foreground/25 mt-6 flex items-center gap-2 font-mono text-[11px]">
                    <IconTerminal2 size={13} />
                    <span>spwn architect start</span>
                </div>
            </div>
        </div>
    );
}
