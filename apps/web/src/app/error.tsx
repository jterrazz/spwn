'use client';

import Link from 'next/link';
import { useEffect } from 'react';

export default function GlobalError({
    error,
    reset,
}: {
    error: Error & { digest?: string };
    reset: () => void;
}) {
    useEffect(() => {
        console.error('Web UI error:', error);
    }, [error]);

    return (
        <div className="flex min-h-screen flex-col items-center justify-center">
            <div className="text-center">
                <div className="mx-auto mb-5 flex h-16 w-16 items-center justify-center rounded-2xl border border-red-500/20 bg-red-500/10">
                    <span className="text-2xl">⚠</span>
                </div>
                <h2 className="font-heading text-foreground/80 mb-2 text-lg">
                    Something went wrong
                </h2>
                <p className="text-muted-foreground/40 mb-6 max-w-md font-mono text-sm">
                    {error.message || 'An unexpected error occurred'}
                </p>
                <div className="flex items-center justify-center gap-3">
                    <button
                        className="text-foreground/60 hover:text-foreground/80 rounded-xl border border-white/[0.06] bg-white/[0.04] px-5 py-2.5 text-sm transition-all hover:bg-white/[0.08]"
                        onClick={reset}
                    >
                        Try again
                    </button>
                    <Link
                        className="text-muted-foreground/40 hover:text-foreground/60 rounded-xl px-5 py-2.5 text-sm transition-colors"
                        href="/"
                    >
                        Go home
                    </Link>
                </div>
            </div>
        </div>
    );
}
