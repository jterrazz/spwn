'use client';

import { IconMessageCircle, IconRocket, IconUser } from '@tabler/icons-react';

import { Skeleton } from '../ui/skeleton';

export type Tab = 'chat' | 'files' | 'profile';

export function AgentLoading() {
    return (
        <div className="max-w-3xl space-y-6 p-8">
            <div className="flex items-center gap-4">
                <Skeleton className="h-12 w-12 rounded-xl" />
                <div className="space-y-2">
                    <Skeleton className="h-6 w-32" />
                    <Skeleton className="h-3 w-48" />
                </div>
            </div>
            <Skeleton className="h-24 w-full rounded-xl" />
            <Skeleton className="h-16 w-full rounded-xl" />
            <div className="flex gap-2">
                {[1, 2, 3, 4].map((index) => (
                    <Skeleton className="h-8 w-24 rounded-full" key={index} />
                ))}
            </div>
            <Skeleton className="h-32 w-full rounded-xl" />
        </div>
    );
}

export function AgentMissing({ agent, onCreate }: { agent: string; onCreate: () => void }) {
    return (
        <div className="flex min-h-[60vh] flex-col items-center justify-center p-8">
            <div className="mb-4 flex h-16 w-16 items-center justify-center rounded-2xl border border-white/[0.06] bg-white/[0.03]">
                <IconUser className="text-muted-foreground/20" size={28} />
            </div>
            <p className="text-muted-foreground/50 font-heading text-lg">
                Agent &quot;{agent}&quot; not found
            </p>
            <p className="text-muted-foreground/30 mt-2 font-mono text-xs">
                Create this agent with: spwn agent create {agent}
            </p>
            <button
                className="text-foreground/60 hover:text-foreground/80 mt-6 flex items-center gap-2 rounded-xl border border-white/[0.06] bg-white/[0.04] px-4 py-2.5 text-sm transition-all hover:bg-white/[0.08]"
                onClick={onCreate}
                type="button"
            >
                <IconRocket size={16} />
                Create &quot;{agent}&quot;
            </button>
        </div>
    );
}

export function TabBar({
    tab,
    files,
    onTab,
}: {
    tab: Tab;
    files: number;
    onTab: (tab: Tab) => void;
}) {
    const tabs: { id: Tab; label: React.ReactNode }[] = [
        {
            id: 'chat',
            label: (
                <>
                    <IconMessageCircle size={13} />
                    Chat
                </>
            ),
        },
        { id: 'profile', label: 'Profile' },
        { id: 'files', label: `Files (${files})` },
    ];
    return (
        <div className="flex gap-1 border-b border-white/[0.06] pb-px">
            {tabs.map((entry) => (
                <button
                    className={`-mb-px flex items-center gap-1.5 border-b-2 px-4 py-2 text-xs font-medium transition-colors ${
                        tab === entry.id
                            ? 'border-foreground/50 text-foreground/80'
                            : 'text-muted-foreground/40 hover:text-muted-foreground/60 border-transparent'
                    }`}
                    key={entry.id}
                    onClick={() => {
                        onTab(entry.id);
                    }}
                    type="button"
                >
                    {entry.label}
                </button>
            ))}
        </div>
    );
}
