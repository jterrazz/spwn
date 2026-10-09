'use client';

import { IconAlertTriangle, IconX } from '@tabler/icons-react';

import { CredentialBadge, metaOf } from './meta';
import type { ProviderInfo } from './meta';
import { Usage } from './usage';

const QUIET =
    'text-muted-foreground/30 hover:text-foreground/60 rounded-lg px-2.5 py-1.5 text-[11px] transition-all hover:bg-white/[0.04]';

interface ProviderRowProps {
    provider: ProviderInfo;
    checking: boolean;
    onConfigure: () => void;
    onReset: () => void;
    onReconnect: () => void;
    onCheck: () => void;
}

function dotTone(provider: ProviderInfo): string {
    if (provider.connected) {
        return 'bg-green-400/70';
    }
    return (provider.error ?? '') === '' ? 'bg-white/15' : 'bg-red-400/70';
}

function Title({ provider }: { provider: ProviderInfo }) {
    const meta = metaOf(provider.provider);
    return (
        <>
            <div
                className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-white/[0.06] bg-white/[0.04] text-base ${meta.color}`}
            >
                {meta.icon}
            </div>
            <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2.5">
                    <span className="text-foreground/80 font-mono text-sm font-medium">
                        {meta.name}
                    </span>
                    <span className={`h-1.5 w-1.5 rounded-full ${dotTone(provider)}`} />
                    <CredentialBadge type={provider.credentialType} />
                </div>
                <div className="mt-0.5 flex items-center gap-2">
                    <span
                        className={`font-mono text-[10px] ${provider.connected ? 'text-muted-foreground/30' : 'text-muted-foreground/20'}`}
                    >
                        {provider.connected ? (provider.source ?? 'Connected') : 'Not configured'}
                    </span>
                </div>
            </div>
        </>
    );
}

function Actions({
    provider,
    checking,
    onConfigure,
    onReset,
    onReconnect,
    onCheck,
}: ProviderRowProps) {
    if (!provider.connected) {
        return (
            <>
                <button
                    className="text-foreground/60 hover:text-foreground/80 flex items-center gap-1.5 rounded-lg border border-white/[0.08] bg-white/[0.05] px-3 py-1.5 text-[11px] font-medium transition-all hover:bg-white/[0.08]"
                    onClick={onReconnect}
                    type="button"
                >
                    Reconnect
                </button>
                <button className={QUIET} onClick={onConfigure} type="button">
                    How to connect
                </button>
            </>
        );
    }
    return (
        <>
            <button
                className={`${QUIET} disabled:opacity-40`}
                disabled={checking}
                onClick={onCheck}
                type="button"
            >
                {checking ? (
                    <span className="border-foreground/30 border-t-foreground/70 inline-block h-3 w-3 animate-spin rounded-full border-2" />
                ) : (
                    'Verify'
                )}
            </button>
            <button className={QUIET} onClick={onConfigure} type="button">
                Change
            </button>
            <button
                aria-label={`Clear ${metaOf(provider.provider).name}`}
                className="text-muted-foreground/20 rounded-lg px-2.5 py-1.5 text-[11px] transition-all hover:bg-red-500/[0.04] hover:text-red-400/60"
                onClick={onReset}
                type="button"
            >
                <IconX size={12} />
            </button>
        </>
    );
}

/** One provider: how it is connected, its plan, its actions, its error, and what it has used. */
export function ProviderRow(props: ProviderRowProps) {
    const { provider } = props;
    const error = provider.error ?? '';
    const plan = provider.plan ?? '';
    return (
        <div className="space-y-3 py-5">
            <div className="flex items-center gap-4">
                <Title provider={provider} />
                {plan !== '' && (
                    <span className="text-muted-foreground/35 rounded-full border border-white/[0.06] bg-white/[0.03] px-2 py-0.5 font-mono text-[9px] tracking-wider uppercase">
                        {plan}
                    </span>
                )}
                <div className="flex shrink-0 items-center gap-1">
                    <Actions {...props} />
                </div>
            </div>
            {error !== '' && (
                <div className="ml-13 flex items-start gap-2 rounded-lg border border-red-500/12 bg-red-500/8 px-3 py-2">
                    <IconAlertTriangle className="mt-0.5 shrink-0 text-red-400/50" size={12} />
                    <p className="font-mono text-[10px] leading-relaxed text-red-400/50">{error}</p>
                </div>
            )}
            {provider.usage !== null && provider.usage !== undefined && (
                <Usage usage={provider.usage} />
            )}
        </div>
    );
}
