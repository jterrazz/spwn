'use client';

import {
    IconAlertTriangle,
    IconCheck,
    IconKey,
    IconLock,
    IconPlugConnected,
    IconTerminal2,
    IconUserCircle,
    IconX,
} from '@tabler/icons-react';
import { useEffect, useState } from 'react';

import { goApiUrl } from '@/api/client';
import { Page } from '@/components/page';
import { PageHeader } from '@/components/page-header';
import { Skeleton } from '@/components/ui/skeleton';
import { usePageTitle } from '@/hooks/use-page-title';

// ── Types ───────────────────────────────────────────────────────────────

interface ProviderUsage {
    session?: { used: number; limit: number; label: string };
    weekly?: { used: number; limit: number; label: string };
    credits?: { used: number; limit: number; currency: string };
}

interface ProviderInfo {
    provider: string;
    connected: boolean;
    credentialType: null | string;
    source: null | string;
    error: null | string;
    plan: null | string;
    usage: null | ProviderUsage;
}

const PROVIDER_META: Record<
    string,
    {
        name: string;
        icon: string;
        color: string;
        docsUrl: string;
        envKey: string;
        oauthNote: string;
    }
> = {
    anthropic: {
        name: 'Anthropic',
        icon: '◆',
        color: 'text-orange-400',
        docsUrl: 'https://console.anthropic.com/settings/keys',
        envKey: 'ANTHROPIC_API_KEY',
        oauthNote: 'Sign in via Claude Code CLI: claude login',
    },
    openai: {
        name: 'OpenAI',
        icon: '◎',
        color: 'text-green-400',
        docsUrl: 'https://platform.openai.com/api-keys',
        envKey: 'OPENAI_API_KEY',
        oauthNote: 'Sign in via Codex CLI: codex login',
    },
    google: {
        name: 'Google',
        icon: '◈',
        color: 'text-blue-400',
        docsUrl: 'https://aistudio.google.com/app/apikey',
        envKey: 'GOOGLE_API_KEY',
        oauthNote: '',
    },
};

// ── Helpers ─────────────────────────────────────────────────────────────

function credLabel(type: null | string): { label: string; icon: React.ReactNode; color: string } {
    switch (type) {
        case 'api_key': {
            return {
                label: 'API Key',
                icon: <IconKey size={10} />,
                color: 'text-blue-400/60 border-blue-500/15 bg-blue-500/8',
            };
        }
        case 'oauth': {
            return {
                label: 'OAuth',
                icon: <IconUserCircle size={10} />,
                color: 'text-purple-400/60 border-purple-500/15 bg-purple-500/8',
            };
        }
        case 'keychain': {
            return {
                label: 'Keychain',
                icon: <IconLock size={10} />,
                color: 'text-amber-400/60 border-amber-500/15 bg-amber-500/8',
            };
        }
        default: {
            return { label: '', icon: null, color: '' };
        }
    }
}

// ── Sub-components ──────────────────────────────────────────────────────

function UsageBar({
    label,
    used,
    limit,
    suffix,
}: {
    label: string;
    used: number;
    limit: number;
    suffix?: string;
}) {
    const pct = limit > 0 ? Math.min((used / limit) * 100, 100) : 0;
    let barColor = 'bg-green-400/70';
    if (pct > 90) {
        barColor = 'bg-red-400';
    } else if (pct > 70) {
        barColor = 'bg-amber-400';
    }
    return (
        <div className="space-y-1">
            <div className="flex items-center justify-between">
                <span className="text-muted-foreground/40 text-[10px]">{label}</span>
                <span className="text-muted-foreground/50 font-mono text-[10px]">
                    {suffix
                        ? `${suffix}${used.toFixed(2)} / ${suffix}${limit.toFixed(2)}`
                        : `${pct.toFixed(0)}%`}
                </span>
            </div>
            <div className="h-1 overflow-hidden rounded-full bg-white/[0.06]">
                <div
                    className={`h-full rounded-full transition-all duration-500 ${barColor}`}
                    style={{ width: `${pct}%` }}
                />
            </div>
        </div>
    );
}

function ProviderRow({
    provider,
    onConfigure,
    onReset,
    onReconnect,
    checking,
    onCheck,
}: {
    provider: ProviderInfo;
    onConfigure: () => void;
    onReset: () => void;
    onReconnect: () => void;
    onCheck: () => void;
    checking: boolean;
}) {
    const meta = PROVIDER_META[provider.provider] ?? {
        name: provider.provider,
        icon: '●',
        color: 'text-white/60',
        docsUrl: '',
        envKey: '',
        oauthNote: '',
    };
    const { connected } = provider;
    const cred = credLabel(provider.credentialType);

    return (
        <div className="space-y-3 py-5">
            {/* Main row */}
            <div className="flex items-center gap-4">
                {/* Icon */}
                <div
                    className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-white/[0.06] bg-white/[0.04] text-base ${meta.color}`}
                >
                    {meta.icon}
                </div>

                {/* Name + status */}
                <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2.5">
                        <span className="text-foreground/80 font-mono text-sm font-medium">
                            {meta.name}
                        </span>
                        {connected && <span className="h-1.5 w-1.5 rounded-full bg-green-400/70" />}
                        {!connected && provider.error && (
                            <span className="h-1.5 w-1.5 rounded-full bg-red-400/70" />
                        )}
                        {!connected && !provider.error && (
                            <span className="h-1.5 w-1.5 rounded-full bg-white/15" />
                        )}
                        {/* Credential type badge */}
                        {cred.label && (
                            <span
                                className={`flex items-center gap-1 rounded border px-1.5 py-0.5 font-mono text-[9px] ${cred.color}`}
                            >
                                {cred.icon}
                                {cred.label}
                            </span>
                        )}
                    </div>
                    <div className="mt-0.5 flex items-center gap-2">
                        {connected ? (
                            <span className="text-muted-foreground/30 font-mono text-[10px]">
                                {provider.source ?? 'Connected'}
                            </span>
                        ) : (
                            <span className="text-muted-foreground/20 font-mono text-[10px]">
                                Not configured
                            </span>
                        )}
                    </div>
                </div>

                {/* Plan badge */}
                {provider.plan && (
                    <span className="text-muted-foreground/35 rounded-full border border-white/[0.06] bg-white/[0.03] px-2 py-0.5 font-mono text-[9px] tracking-wider uppercase">
                        {provider.plan}
                    </span>
                )}

                {/* Actions */}
                <div className="flex shrink-0 items-center gap-1">
                    {!connected && (
                        <>
                            <button
                                className="text-foreground/60 hover:text-foreground/80 flex items-center gap-1.5 rounded-lg border border-white/[0.08] bg-white/[0.05] px-3 py-1.5 text-[11px] font-medium transition-all hover:bg-white/[0.08]"
                                onClick={onReconnect}
                            >
                                Reconnect
                            </button>
                            <button
                                className="text-muted-foreground/30 hover:text-foreground/60 rounded-lg px-2.5 py-1.5 text-[11px] transition-all hover:bg-white/[0.04]"
                                onClick={onConfigure}
                            >
                                How to connect
                            </button>
                        </>
                    )}
                    {connected && (
                        <>
                            <button
                                className="text-muted-foreground/30 hover:text-foreground/60 rounded-lg px-2.5 py-1.5 text-[11px] transition-all hover:bg-white/[0.04] disabled:opacity-40"
                                disabled={checking}
                                onClick={onCheck}
                            >
                                {checking ? (
                                    <span className="border-foreground/30 border-t-foreground/70 inline-block h-3 w-3 animate-spin rounded-full border-2" />
                                ) : (
                                    'Verify'
                                )}
                            </button>
                            <button
                                className="text-muted-foreground/30 hover:text-foreground/60 rounded-lg px-2.5 py-1.5 text-[11px] transition-all hover:bg-white/[0.04]"
                                onClick={onConfigure}
                            >
                                Change
                            </button>
                            <button
                                className="text-muted-foreground/20 rounded-lg px-2.5 py-1.5 text-[11px] transition-all hover:bg-red-500/[0.04] hover:text-red-400/60"
                                onClick={onReset}
                            >
                                <IconX size={12} />
                            </button>
                        </>
                    )}
                </div>
            </div>

            {/* Error */}
            {provider.error && (
                <div className="ml-13 flex items-start gap-2 rounded-lg border border-red-500/12 bg-red-500/8 px-3 py-2">
                    <IconAlertTriangle className="mt-0.5 shrink-0 text-red-400/50" size={12} />
                    <p className="font-mono text-[10px] leading-relaxed text-red-400/50">
                        {provider.error}
                    </p>
                </div>
            )}

            {/* Usage bars */}
            {provider.usage && (
                <div className="ml-13 max-w-xs space-y-2">
                    {provider.usage.session && (
                        <UsageBar
                            label={`Session (${provider.usage.session.label})`}
                            limit={provider.usage.session.limit}
                            used={provider.usage.session.used}
                        />
                    )}
                    {provider.usage.weekly && (
                        <UsageBar
                            label={`Weekly (${provider.usage.weekly.label})`}
                            limit={provider.usage.weekly.limit}
                            used={provider.usage.weekly.used}
                        />
                    )}
                    {provider.usage.credits && (
                        <UsageBar
                            label="Credits"
                            limit={provider.usage.credits.limit}
                            suffix={provider.usage.credits.currency}
                            used={provider.usage.credits.used}
                        />
                    )}
                </div>
            )}
        </div>
    );
}

function ConfigureModal({ provider, onClose }: { provider: string; onClose: () => void }) {
    const meta = PROVIDER_META[provider];

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
            <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={onClose} />
            <div className="bg-popover/95 relative z-10 mx-4 w-full max-w-md rounded-2xl border border-white/[0.08] p-6 shadow-2xl backdrop-blur-md">
                <div className="mb-5 flex items-center justify-between">
                    <h3 className="font-heading text-foreground/80 text-sm tracking-wide">
                        Connect {meta?.name ?? provider}
                    </h3>
                    <button
                        className="text-muted-foreground/30 hover:text-muted-foreground/60 p-1 transition-colors"
                        onClick={onClose}
                    >
                        <IconX size={16} />
                    </button>
                </div>

                <div className="space-y-4">
                    <div className="space-y-3 rounded-lg border border-white/[0.06] bg-white/[0.02] p-4">
                        <p className="text-muted-foreground/50 text-xs leading-relaxed">
                            Sign in via the runtime CLI on your host machine. Spwn detects
                            credentials from your system keychain and shares them with all
                            containers automatically.
                        </p>
                        {meta?.oauthNote && (
                            <div className="text-foreground/50 flex items-center gap-2 rounded-lg border border-white/[0.06] bg-white/[0.03] px-3 py-2 font-mono text-[11px]">
                                <IconTerminal2
                                    className="text-muted-foreground/30 shrink-0"
                                    size={13}
                                />
                                <span>{meta.oauthNote}</span>
                            </div>
                        )}
                        <p className="text-muted-foreground/25 text-[10px]">
                            After signing in, click Reconnect in the settings to pick up the new
                            credentials.
                        </p>
                    </div>

                    <button
                        className="text-muted-foreground/50 hover:text-foreground/70 w-full rounded-xl px-4 py-2.5 text-sm transition-colors hover:bg-white/[0.04]"
                        onClick={onClose}
                    >
                        Done
                    </button>
                </div>
            </div>
        </div>
    );
}

// ── Page ─────────────────────────────────────────────────────────────────

export default function ProvidersPage() {
    usePageTitle('Settings');
    const [providers, setProviders] = useState<ProviderInfo[]>([]);
    const [loading, setLoading] = useState(true);
    const [errorMessage, setErrorMessage] = useState<null | string>(null);
    const [checking, setChecking] = useState<null | string>(null);
    const [configuring, setConfiguring] = useState<null | string>(null);
    const [feedback, setFeedback] = useState<null | { message: string; type: 'error' | 'success' }>(
        null,
    );

    const showFeedback = (message: string, type: 'error' | 'success') => {
        setFeedback({ message, type });
        setTimeout(() => setFeedback(null), 3000);
    };

    const fetchProviders = async () => {
        try {
            const res = await fetch(goApiUrl('/api/auth/providers'));
            if (!res.ok) {
                throw new Error('Failed to fetch providers');
            }
            const data = await res.json();
            setProviders(data.providers ?? []);
            setErrorMessage(null);
        } catch (error) {
            setErrorMessage(error instanceof Error ? error.message : 'Failed to load providers');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchProviders();
    }, []);

    const handleCheck = async (providerName: string) => {
        setChecking(providerName);
        try {
            const res = await fetch(goApiUrl('/api/auth/check'), {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ provider: providerName }),
            });
            const data = await res.json();
            if (data.connected) {
                showFeedback(`${providerName} verified`, 'success');
                setProviders((prev) =>
                    prev.map((p) =>
                        p.provider === providerName
                            ? { ...p, connected: true, error: null, usage: data.usage }
                            : p,
                    ),
                );
            } else {
                showFeedback(data.error || `${providerName} failed`, 'error');
                setProviders((prev) =>
                    prev.map((p) =>
                        p.provider === providerName
                            ? { ...p, connected: false, error: data.error }
                            : p,
                    ),
                );
            }
        } catch {
            showFeedback(`Failed to check ${providerName}`, 'error');
        } finally {
            setChecking(null);
        }
    };

    const handleReset = async (providerName: string) => {
        try {
            const res = await fetch(goApiUrl('/api/auth/reset'), {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ provider: providerName }),
            });
            if (res.ok) {
                showFeedback(`${providerName} cleared`, 'success');
                fetchProviders();
            } else {
                showFeedback('Failed to reset', 'error');
            }
        } catch {
            showFeedback('Failed to reset', 'error');
        }
    };

    const handleReconnect = async (providerName: string) => {
        try {
            const res = await fetch(goApiUrl('/api/auth/reconnect'), {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ provider: providerName }),
            });
            if (res.ok) {
                showFeedback(`${providerName} reconnected`, 'success');
                fetchProviders();
            } else {
                showFeedback('Failed to reconnect', 'error');
            }
        } catch {
            showFeedback('Failed to reconnect', 'error');
        }
    };

    return (
        <Page>
            <PageHeader description="Connect your AI providers to power agents." title="Settings" />

            {/* Feedback toast */}
            {feedback && (
                <div
                    className={`animate-in fade-in slide-in-from-top-2 rounded-lg px-4 py-2 font-mono text-xs duration-200 ${
                        feedback.type === 'success'
                            ? 'border border-green-500/20 bg-green-500/10 text-green-400'
                            : 'border border-red-500/20 bg-red-500/10 text-red-400'
                    }`}
                >
                    {feedback.type === 'success' ? (
                        <IconCheck className="mr-1.5 inline" size={12} />
                    ) : (
                        <IconAlertTriangle className="mr-1.5 inline" size={12} />
                    )}
                    {feedback.message}
                </div>
            )}

            {/* Error */}
            {errorMessage && !loading && (
                <div className="flex items-start gap-2 rounded-lg border border-red-500/15 bg-red-500/10 px-4 py-3">
                    <IconAlertTriangle className="mt-0.5 shrink-0 text-red-400/60" size={14} />
                    <p className="font-mono text-xs text-red-400/70">{errorMessage}</p>
                </div>
            )}

            {/* Provider list */}
            {loading && (
                <div className="space-y-4">
                    {[1, 2, 3].map((i) => (
                        <div className="flex items-center gap-4 py-5" key={i}>
                            <Skeleton className="h-9 w-9 rounded-lg" />
                            <div className="flex-1">
                                <Skeleton className="h-4 w-28" />
                                <Skeleton className="mt-1.5 h-2.5 w-20" />
                            </div>
                            <Skeleton className="h-7 w-20 rounded-lg" />
                        </div>
                    ))}
                </div>
            )}
            {!loading && providers.length === 0 && !errorMessage && (
                <div className="-mt-12 flex flex-1 items-center justify-center">
                    <div className="flex max-w-md flex-col items-center text-center">
                        <div className="mb-6 flex h-16 w-16 items-center justify-center rounded-2xl border border-white/[0.08] bg-white/[0.04]">
                            <IconPlugConnected className="text-muted-foreground/20" size={28} />
                        </div>
                        <h2 className="font-heading text-foreground/70 mb-2 text-lg tracking-wide">
                            No providers detected
                        </h2>
                        <p className="text-muted-foreground/40 mb-6 text-sm leading-relaxed">
                            Agents need an AI provider to think. Add an API key or sign in with a
                            subscription.
                        </p>
                        <div className="text-muted-foreground/25 flex items-center gap-2 font-mono text-[11px]">
                            <IconTerminal2 size={13} />
                            <span>export ANTHROPIC_API_KEY=sk-...</span>
                        </div>
                    </div>
                </div>
            )}
            {!loading && providers.length > 0 && (
                <div className="divide-y divide-white/[0.06]">
                    {providers.map((provider) => (
                        <ProviderRow
                            checking={checking === provider.provider}
                            key={provider.provider}
                            onCheck={async () => await handleCheck(provider.provider)}
                            onConfigure={() => setConfiguring(provider.provider)}
                            onReconnect={async () => await handleReconnect(provider.provider)}
                            onReset={async () => await handleReset(provider.provider)}
                            provider={provider}
                        />
                    ))}
                </div>
            )}

            {/* Configure Modal */}
            {configuring && (
                <ConfigureModal onClose={() => setConfiguring(null)} provider={configuring} />
            )}
        </Page>
    );
}
