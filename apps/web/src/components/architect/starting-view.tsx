'use client';

import { IconHexagonFilled } from '@tabler/icons-react';
import { useState } from 'react';

import { ProgressShimmer } from '../progress-shimmer';
import { describeProgress } from './progress';
import type { SpawnProgress } from './progress';

function Glyph({ failed }: { failed: boolean }) {
    return (
        <div
            className={`relative mb-6 flex h-16 w-16 items-center justify-center rounded-2xl border ${failed ? 'border-red-400/30 bg-red-500/[0.06]' : 'border-white/[0.08] bg-white/[0.04]'}`}
        >
            <IconHexagonFilled
                className={failed ? 'text-red-400/60' : 'animate-pulse text-yellow-400/40'}
                size={28}
            />
        </div>
    );
}

function Logs({ progress }: { progress: null | SpawnProgress }) {
    const [shown, setShown] = useState(false);
    const tail = progress?.logTail ?? '';
    return (
        <>
            <div className="text-muted-foreground/40 flex items-center gap-2 text-[10px] tracking-wider uppercase">
                <span>{progress?.elapsedSeconds ?? 0}s elapsed</span>
                <span className="opacity-30">·</span>
                <button
                    className="hover:text-foreground/80 transition-colors"
                    onClick={() => {
                        setShown(!shown);
                    }}
                    type="button"
                >
                    {shown ? 'Hide logs' : 'View logs'}
                </button>
            </div>
            {shown && tail !== '' && (
                <pre className="text-muted-foreground/70 mt-4 max-h-64 w-full overflow-auto rounded-lg border border-white/[0.06] bg-black/40 px-3 py-2 text-left font-mono text-[10px] leading-snug">
                    {tail}
                </pre>
            )}
        </>
    );
}

/** The architect page while the architect starts: the step its start reports, its time, its logs, or why it failed. */
export function StartingView({ progress }: { progress: null | SpawnProgress }) {
    const error = progress?.error ?? '';
    const failed = error !== '';
    return (
        <div className="-mt-12 flex flex-1 items-center justify-center px-4">
            <div className="flex w-full max-w-lg flex-col items-center text-center">
                <Glyph failed={failed} />
                <h2 className="font-heading text-foreground/70 mb-2 text-lg tracking-wide">
                    {failed ? 'Architect failed to start' : 'Starting Architect'}
                </h2>
                <p
                    className={`mb-4 text-sm leading-relaxed ${failed ? 'text-red-300/70' : 'text-muted-foreground/50'}`}
                >
                    {failed ? error : describeProgress(progress)}
                </p>
                {!failed && (
                    <div className="mb-3 w-48">
                        <ProgressShimmer active message="" />
                    </div>
                )}
                <Logs progress={progress} />
            </div>
        </div>
    );
}
