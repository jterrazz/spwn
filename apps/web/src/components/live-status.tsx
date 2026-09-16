'use client';

import { useEffect, useState } from 'react';

import { getConnectionStatus, isGoApiAvailable, onConnectionStatusChange } from '@/api/client';
import type { ConnectionStatus } from '@/api/client';

const STATUS_CONFIG: Record<
    ConnectionStatus,
    { dot: string; label: string; labelColor: string; title: string }
> = {
    connected: {
        dot: 'bg-green-500 shadow-[0_0_4px_rgba(34,197,94,0.6)]',
        label: 'Connected',
        labelColor: 'text-green-500/60',
        title: 'Connected - Go API responding',
    },
    disconnected: {
        dot: 'bg-red-400 shadow-[0_0_4px_rgba(248,113,113,0.5)]',
        label: 'Disconnected',
        labelColor: 'text-red-400/60',
        title: 'Disconnected - API unreachable',
    },
};

export function LiveStatus() {
    const [status, setStatus] = useState<ConnectionStatus>(getConnectionStatus());

    useEffect(() => {
        const unsub = onConnectionStatusChange(setStatus);

        const check = async () => {
            const goUp = await isGoApiAvailable();
            setStatus(goUp ? 'connected' : 'disconnected');
        };

        check();
        const interval = setInterval(check, 10_000);
        return () => {
            clearInterval(interval);
            unsub();
        };
    }, []);

    const config = STATUS_CONFIG[status];

    return (
        <div
            className="bg-foreground/[0.06] border-foreground/[0.08] flex h-8 items-center gap-1.5 rounded-full border px-2.5 shadow-[inset_0_1px_0_rgba(255,255,255,0.12),0_1px_2px_rgba(0,0,0,0.05)] backdrop-blur-md dark:border-white/[0.1] dark:bg-white/[0.08] dark:shadow-[inset_0_1px_0_rgba(255,255,255,0.06),0_1px_2px_rgba(0,0,0,0.2)]"
            title={config.title}
        >
            <div className={`h-2 w-2 rounded-full transition-colors ${config.dot}`} />
            <span className={`font-mono text-[10px] tracking-wider uppercase ${config.labelColor}`}>
                {config.label}
            </span>
        </div>
    );
}
