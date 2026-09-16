'use client';

import { IconArrowUp, IconX } from '@tabler/icons-react';
import { useEffect, useState } from 'react';

import type { VersionInfo } from '@/hooks/use-version';

const DISMISS_KEY = 'spwn-upgrade-dismissed';

interface UpgradeBannerProps {
    version: VersionInfo;
}

export function UpgradeBanner({ version }: UpgradeBannerProps) {
    const [dismissed, setDismissed] = useState(true); // Start hidden to avoid flash

    useEffect(() => {
        // Check if this specific version was already dismissed
        const dismissedVersion = localStorage.getItem(DISMISS_KEY);
        setDismissed(dismissedVersion === version.latest);
    }, [version.latest]);

    if (!version.updateAvailable || dismissed) {
        return null;
    }

    const handleDismiss = () => {
        localStorage.setItem(DISMISS_KEY, version.latest);
        setDismissed(true);
    };

    return (
        <div className="mx-3 mb-3 rounded-lg border border-amber-500/20 bg-amber-500/[0.06] p-3">
            <div className="flex items-start justify-between gap-2">
                <div className="flex items-center gap-2 text-amber-400">
                    <IconArrowUp className="mt-0.5 shrink-0" size={14} />
                    <span className="text-[11px] font-medium">
                        spwn v{version.latest} available
                    </span>
                </div>
                <button
                    className="text-muted-foreground/30 hover:text-muted-foreground/60 transition-colors"
                    onClick={handleDismiss}
                >
                    <IconX size={12} />
                </button>
            </div>
            <p className="text-muted-foreground/50 mt-1.5 ml-[22px] text-[10px]">
                Current: v{version.current}
            </p>
            <div className="mt-2 ml-[22px] space-y-1">
                <p className="text-muted-foreground/40 text-[10px]">Run in terminal:</p>
                <code className="block rounded bg-black/20 px-2 py-1 font-mono text-[10px] text-amber-400/70">
                    $ spwn upgrade
                </code>
                <p className="text-muted-foreground/40 mt-1.5 text-[10px]">Then restart:</p>
                <code className="block rounded bg-black/20 px-2 py-1 font-mono text-[10px] text-amber-400/70">
                    $ spwn web
                </code>
            </div>
        </div>
    );
}
