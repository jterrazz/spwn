'use client';

import { useCallback, useState } from 'react';

import { apiAction, apiGet } from '../../api/client';
import { inBackground } from '../../events/in-background';
import { useBackgroundTask } from '../../hooks/use-background-task';
import { Continue } from './step-views';
import type { OnboardingStatus } from './steps';

// value: a provider connected in another window shows up here within this many milliseconds
const PROVIDER_POLL_MS = 4000;

interface ProviderRow {
    provider: string;
    connected: boolean;
    source?: string;
}

function useProviders() {
    const [providers, setProviders] = useState<null | ProviderRow[]>(null);
    const load = useCallback(async () => {
        try {
            const listed = await apiGet<{ providers?: ProviderRow[] }>('/api/auth/providers');
            setProviders(listed.providers ?? []);
        } catch {
            setProviders([]);
        }
    }, []);
    useBackgroundTask(load, PROVIDER_POLL_MS);
    return providers;
}

function ProviderLine({ row }: { row: ProviderRow }) {
    const connect = async () => {
        await apiAction('/api/auth/reconnect', { provider: row.provider });
    };
    return (
        <div className="flex items-center justify-between rounded-md border border-white/[0.06] bg-white/[0.015] px-3 py-2">
            <div className="flex items-center gap-2">
                <div
                    className={`h-1.5 w-1.5 rounded-full ${row.connected ? 'bg-emerald-400' : 'bg-muted-foreground/30'}`}
                />
                <span className="text-foreground/90 text-xs font-medium capitalize">
                    {row.provider}
                </span>
                {row.source !== undefined && row.source !== '' && (
                    <span className="text-muted-foreground/40 text-[10px]">{row.source}</span>
                )}
            </div>
            {row.connected ? (
                <span className="text-[10px] tracking-wider text-emerald-300/80 uppercase">
                    Connected
                </span>
            ) : (
                <button
                    className="text-foreground/70 hover:text-foreground text-[11px]"
                    onClick={inBackground(connect)}
                    type="button"
                >
                    Connect
                </button>
            )}
        </div>
    );
}

export function AuthenticationStep({
    status,
    onAdvance,
}: {
    status: null | OnboardingStatus;
    onAdvance: () => void;
}) {
    const providers = useProviders();
    return (
        <div className="space-y-3">
            <p className="text-muted-foreground/60 text-[11px]">
                spwn supports subscription sign-in (Anthropic / Codex) and API keys via env vars.
            </p>
            <div className="space-y-1.5">
                {providers === null && (
                    <div className="text-muted-foreground/50 rounded-md border border-white/[0.06] bg-white/[0.015] px-3 py-2 text-[11px]">
                        Loading providers...
                    </div>
                )}
                {providers?.map((row) => (
                    <ProviderLine key={row.provider} row={row} />
                ))}
            </div>
            {status?.hasAuth === true && <Continue onAdvance={onAdvance} />}
        </div>
    );
}
