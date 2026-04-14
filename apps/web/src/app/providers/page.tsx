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

import { Page } from '@/components/page';
import { PageHeader } from '@/components/page-header';
import { Skeleton } from '@/components/ui/skeleton';
import { usePageTitle } from '@/hooks/use-page-title';
import { goApiUrl } from '@/lib/api-client';

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
    const barColor = pct > 90 ? 'bg-red-400' : pct > 70 ? 'bg-amber-400' : 'bg-green-400/70';
    return (
        <div className="space-y-1">
            <div className="flex items-center justify-between">
                <span className="text-[10px] text-muted-foreground/40">{label}</span>
                <span className="text-[10px] font-mono text-muted-foreground/50">
                    {suffix
                        ? `${suffix}${used.toFixed(2)} / ${suffix}${limit.toFixed(2)}`
                        : `${pct.toFixed(0)}%`}
                </span>
            </div>
            <div className="h-1 rounded-full bg-white/[0.06] overflow-hidden">
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
    const connected = provider.connected;
    const cred = credLabel(provider.credentialType);

    return (
        <div className="py-5 space-y-3">
            {/* Main row */}
            <div className="flex items-center gap-4">
                {/* Icon */}
                <div
                    className={`w-9 h-9 rounded-lg bg-white/[0.04] border border-white/[0.06] flex items-center justify-center text-base shrink-0 ${meta.color}`}
                >
                    {meta.icon}
                </div>

                {/* Name + status */}
                <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2.5">
                        <span className="text-sm font-mono font-medium text-foreground/80">
                            {meta.name}
                        </span>
                        {connected ? (
                            <span className="w-1.5 h-1.5 rounded-full bg-green-400/70" />
                        ) : provider.error ? (
                            <span className="w-1.5 h-1.5 rounded-full bg-red-400/70" />
                        ) : (
                            <span className="w-1.5 h-1.5 rounded-full bg-white/15" />
                        )}
                        {/* Credential type badge */}
                        {cred.label && (
                            <span
                                className={`flex items-center gap-1 text-[9px] font-mono px-1.5 py-0.5 rounded border ${cred.color}`}
                            >
                                {cred.icon}
                                {cred.label}
                            </span>
                        )}
                    </div>
                    <div className="flex items-center gap-2 mt-0.5">
                        {connected ? (
                            <span className="text-[10px] font-mono text-muted-foreground/30">
                                {provider.source ?? 'Connected'}
                            </span>
                        ) : (
                            <span className="text-[10px] font-mono text-muted-foreground/20">
                                Not configured
                            </span>
                        )}
                    </div>
                </div>

                {/* Plan badge */}
                {provider.plan && (
                    <span className="text-[9px] font-mono uppercase tracking-wider px-2 py-0.5 rounded-full border bg-white/[0.03] text-muted-foreground/35 border-white/[0.06]">
                        {provider.plan}
                    </span>
                )}

                {/* Actions */}
                <div className="flex items-center gap-1 shrink-0">
                    {!connected && (
                        <>
                            <button
                                className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-[11px] font-medium bg-white/[0.05] text-foreground/60 hover:text-foreground/80 hover:bg-white/[0.08] border border-white/[0.08] transition-all"
                                onClick={onReconnect}
                            >
                                Reconnect
                            </button>
                            <button
                                className="px-2.5 py-1.5 rounded-lg text-[11px] text-muted-foreground/30 hover:text-foreground/60 hover:bg-white/[0.04] transition-all"
                                onClick={onConfigure}
                            >
                                How to connect
                            </button>
                        </>
                    )}
                    {connected && (
                        <>
                            <button
                                className="px-2.5 py-1.5 rounded-lg text-[11px] text-muted-foreground/30 hover:text-foreground/60 hover:bg-white/[0.04] transition-all disabled:opacity-40"
                                disabled={checking}
                                onClick={onCheck}
                            >
                                {checking ? (
                                    <span className="w-3 h-3 border-2 border-foreground/30 border-t-foreground/70 rounded-full animate-spin inline-block" />
                                ) : (
                                    'Verify'
                                )}
                            </button>
                            <button
                                className="px-2.5 py-1.5 rounded-lg text-[11px] text-muted-foreground/30 hover:text-foreground/60 hover:bg-white/[0.04] transition-all"
                                onClick={onConfigure}
                            >
                                Change
                            </button>
                            <button
                                className="px-2.5 py-1.5 rounded-lg text-[11px] text-muted-foreground/20 hover:text-red-400/60 hover:bg-red-500/[0.04] transition-all"
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
                <div className="ml-13 rounded-lg bg-red-500/8 border border-red-500/12 px-3 py-2 flex items-start gap-2">
                    <IconAlertTriangle className="text-red-400/50 mt-0.5 shrink-0" size={12} />
                    <p className="text-[10px] text-red-400/50 font-mono leading-relaxed">
                        {provider.error}
                    </p>
                </div>
            )}

            {/* Usage bars */}
            {provider.usage && (
                <div className="ml-13 space-y-2 max-w-xs">
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
            <div className="relative z-10 w-full max-w-md mx-4 rounded-2xl bg-popover/95 backdrop-blur-md border border-white/[0.08] shadow-2xl p-6">
                <div className="flex items-center justify-between mb-5">
                    <h3 className="text-sm font-heading tracking-wide text-foreground/80">
                        Connect {meta?.name ?? provider}
                    </h3>
                    <button
                        className="p-1 text-muted-foreground/30 hover:text-muted-foreground/60 transition-colors"
                        onClick={onClose}
                    >
                        <IconX size={16} />
                    </button>
                </div>

                <div className="space-y-4">
                    <div className="rounded-lg border border-white/[0.06] bg-white/[0.02] p-4 space-y-3">
                        <p className="text-xs text-muted-foreground/50 leading-relaxed">
                            Sign in via the runtime CLI on your host machine. Spwn detects
                            credentials from your system keychain and shares them with all
                            containers automatically.
                        </p>
                        {meta?.oauthNote && (
                            <div className="flex items-center gap-2 text-[11px] font-mono text-foreground/50 bg-white/[0.03] border border-white/[0.06] rounded-lg px-3 py-2">
                                <IconTerminal2
                                    className="text-muted-foreground/30 shrink-0"
                                    size={13}
                                />
                                <span>{meta.oauthNote}</span>
                            </div>
                        )}
                        <p className="text-[10px] text-muted-foreground/25">
                            After signing in, click Reconnect in the settings to pick up the new
                            credentials.
                        </p>
                    </div>

                    <button
                        className="w-full px-4 py-2.5 rounded-xl text-sm text-muted-foreground/50 hover:text-foreground/70 hover:bg-white/[0.04] transition-colors"
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
    const [error, setError] = useState<null | string>(null);
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
            setError(null);
        } catch (error) {
            setError(error instanceof Error ? error.message : 'Failed to load providers');
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
                    className={`px-4 py-2 rounded-lg text-xs font-mono animate-in fade-in slide-in-from-top-2 duration-200 ${
                        feedback.type === 'success'
                            ? 'bg-green-500/10 border border-green-500/20 text-green-400'
                            : 'bg-red-500/10 border border-red-500/20 text-red-400'
                    }`}
                >
                    {feedback.type === 'success' ? (
                        <IconCheck className="inline mr-1.5" size={12} />
                    ) : (
                        <IconAlertTriangle className="inline mr-1.5" size={12} />
                    )}
                    {feedback.message}
                </div>
            )}

            {/* Error */}
            {error && !loading && (
                <div className="rounded-lg bg-red-500/10 border border-red-500/15 px-4 py-3 flex items-start gap-2">
                    <IconAlertTriangle className="text-red-400/60 mt-0.5 shrink-0" size={14} />
                    <p className="text-xs text-red-400/70 font-mono">{error}</p>
                </div>
            )}

            {/* Provider list */}
            {loading ? (
                <div className="space-y-4">
                    {[1, 2, 3].map((i) => (
                        <div className="flex items-center gap-4 py-5" key={i}>
                            <Skeleton className="w-9 h-9 rounded-lg" />
                            <div className="flex-1">
                                <Skeleton className="h-4 w-28" />
                                <Skeleton className="h-2.5 w-20 mt-1.5" />
                            </div>
                            <Skeleton className="h-7 w-20 rounded-lg" />
                        </div>
                    ))}
                </div>
            ) : providers.length === 0 && !error ? (
                <div className="flex-1 flex items-center justify-center -mt-12">
                    <div className="flex flex-col items-center text-center max-w-md">
                        <div className="w-16 h-16 rounded-2xl bg-white/[0.04] border border-white/[0.08] flex items-center justify-center mb-6">
                            <IconPlugConnected className="text-muted-foreground/20" size={28} />
                        </div>
                        <h2 className="text-lg font-heading tracking-wide text-foreground/70 mb-2">
                            No providers detected
                        </h2>
                        <p className="text-sm text-muted-foreground/40 mb-6 leading-relaxed">
                            Agents need an AI provider to think. Add an API key or sign in with a
                            subscription.
                        </p>
                        <div className="flex items-center gap-2 text-[11px] text-muted-foreground/25 font-mono">
                            <IconTerminal2 size={13} />
                            <span>export ANTHROPIC_API_KEY=sk-...</span>
                        </div>
                    </div>
                </div>
            ) : (
                <div className="divide-y divide-white/[0.06]">
                    {providers.map((provider) => (
                        <ProviderRow
                            checking={checking === provider.provider}
                            key={provider.provider}
                            onCheck={() => handleCheck(provider.provider)}
                            onConfigure={() => setConfiguring(provider.provider)}
                            onReconnect={() => handleReconnect(provider.provider)}
                            onReset={() => handleReset(provider.provider)}
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
