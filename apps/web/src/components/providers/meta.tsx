import { IconKey, IconLock, IconUserCircle } from '@tabler/icons-react';

export interface ProviderUsage {
    session?: { used: number; limit: number; label: string };
    weekly?: { used: number; limit: number; label: string };
    credits?: { used: number; limit: number; currency: string };
}

// outside: the Go API leaves out the fields a provider has no value for
export interface ProviderInfo {
    provider: string;
    connected: boolean;
    credentialType?: null | string;
    source?: null | string;
    error?: null | string;
    plan?: null | string;
    usage?: null | ProviderUsage;
}

interface Meta {
    name: string;
    icon: string;
    color: string;
    signIn: string;
}

const META: Record<string, Meta> = {
    anthropic: {
        color: 'text-orange-400',
        icon: '◆',
        name: 'Anthropic',
        signIn: 'Sign in via Claude Code CLI: claude login',
    },
    google: { color: 'text-blue-400', icon: '◈', name: 'Google', signIn: '' },
    openai: {
        color: 'text-green-400',
        icon: '◎',
        name: 'OpenAI',
        signIn: 'Sign in via Codex CLI: codex login',
    },
};

export function metaOf(provider: string): Meta {
    return META[provider] ?? { color: 'text-white/60', icon: '●', name: provider, signIn: '' };
}

const CREDENTIALS: Record<string, { label: string; icon: React.ReactNode; color: string }> = {
    api_key: {
        color: 'text-blue-400/60 border-blue-500/15 bg-blue-500/8',
        icon: <IconKey size={10} />,
        label: 'API Key',
    },
    keychain: {
        color: 'text-amber-400/60 border-amber-500/15 bg-amber-500/8',
        icon: <IconLock size={10} />,
        label: 'Keychain',
    },
    oauth: {
        color: 'text-purple-400/60 border-purple-500/15 bg-purple-500/8',
        icon: <IconUserCircle size={10} />,
        label: 'OAuth',
    },
};

export function CredentialBadge({ type }: { type: null | string | undefined }) {
    const credential = type === null || type === undefined ? undefined : CREDENTIALS[type];
    if (credential === undefined) {
        return null;
    }
    return (
        <span
            className={`flex items-center gap-1 rounded border px-1.5 py-0.5 font-mono text-[9px] ${credential.color}`}
        >
            {credential.icon}
            {credential.label}
        </span>
    );
}
