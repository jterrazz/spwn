'use client';

import {
    IconBrain,
    IconCamera,
    IconGitFork,
    IconHexagonFilled,
    IconHexagonOff,
    IconMessageFilled,
    IconMoonFilled,
    IconSparkles,
    IconTrashFilled,
    IconUserFilled,
    IconUserOff,
    IconWorldFilled,
} from '@tabler/icons-react';
import { useEffect, useState } from 'react';

import { apiGet } from '@/api/client';

type ActivityType =
    | 'agent.created'
    | 'agent.deleted'
    | 'agent.dreamed'
    | 'agent.forked'
    | 'agent.joined'
    | 'agent.left'
    | 'agent.slept'
    | 'agent.talked'
    | 'architect.started'
    | 'architect.stopped'
    | 'architect.talked'
    | 'world.destroyed'
    | 'world.session_ended'
    | 'world.snapshot'
    | 'world.spawned'
    | 'world.state_changed';

interface ActivityEvent {
    id: string;
    timestamp: string;
    type: ActivityType;
    actor: string;
    verb: string;
    target?: string;
    phrase: string;
    world_id?: string;
    agent_id?: string;
    duration_ms?: number;
    cost_usd?: number;
}

function timeAgo(iso: string): string {
    const d = Date.now() - new Date(iso).getTime();
    if (d < 0) {
        return 'just now';
    }
    const s = Math.floor(d / 1000);
    if (s < 60) {
        return 'just now';
    }
    const m = Math.floor(s / 60);
    if (m < 60) {
        return `${m}m ago`;
    }
    const h = Math.floor(m / 60);
    if (h < 24) {
        return `${h}h ago`;
    }
    return `${Math.floor(h / 24)}d ago`;
}

const TYPE_CONFIG: Record<
    ActivityType,
    { icon: typeof IconWorldFilled; color: string; bg: string }
> = {
    'world.spawned': {
        icon: IconWorldFilled,
        color: 'text-green-400/70',
        bg: 'bg-green-500/[0.08]',
    },
    'world.destroyed': { icon: IconTrashFilled, color: 'text-red-400/70', bg: 'bg-red-500/[0.08]' },
    'world.snapshot': { icon: IconCamera, color: 'text-blue-400/70', bg: 'bg-blue-500/[0.08]' },
    'world.state_changed': {
        icon: IconSparkles,
        color: 'text-amber-400/70',
        bg: 'bg-amber-500/[0.08]',
    },
    'world.session_ended': {
        icon: IconSparkles,
        color: 'text-foreground/50',
        bg: 'bg-white/[0.04]',
    },
    'agent.created': { icon: IconUserFilled, color: 'text-blue-400/70', bg: 'bg-blue-500/[0.08]' },
    'agent.deleted': { icon: IconUserOff, color: 'text-red-400/70', bg: 'bg-red-500/[0.08]' },
    'agent.joined': { icon: IconUserFilled, color: 'text-blue-400/70', bg: 'bg-blue-500/[0.08]' },
    'agent.left': { icon: IconUserOff, color: 'text-zinc-400/60', bg: 'bg-zinc-500/[0.08]' },
    'agent.dreamed': { icon: IconBrain, color: 'text-purple-400/70', bg: 'bg-purple-500/[0.08]' },
    'agent.slept': {
        icon: IconMoonFilled,
        color: 'text-purple-400/70',
        bg: 'bg-purple-500/[0.08]',
    },
    'agent.forked': { icon: IconGitFork, color: 'text-cyan-400/70', bg: 'bg-cyan-500/[0.08]' },
    'agent.talked': { icon: IconMessageFilled, color: 'text-foreground/50', bg: 'bg-white/[0.04]' },
    'architect.started': {
        icon: IconHexagonFilled,
        color: 'text-green-400/70',
        bg: 'bg-green-500/[0.08]',
    },
    'architect.stopped': {
        icon: IconHexagonOff,
        color: 'text-zinc-400/60',
        bg: 'bg-zinc-500/[0.08]',
    },
    'architect.talked': {
        icon: IconHexagonFilled,
        color: 'text-foreground/50',
        bg: 'bg-white/[0.04]',
    },
};

export function RecentActivity() {
    const [events, setEvents] = useState<ActivityEvent[]>([]);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        const fetchEvents = async () => {
            try {
                const data = await apiGet<{ events: ActivityEvent[] }>('/api/activity?limit=12');
                setEvents(data.events ?? []);
            } catch {
                // Keep old events
            } finally {
                setLoading(false);
            }
        };
        fetchEvents();
        const interval = setInterval(fetchEvents, 10_000);
        return () => clearInterval(interval);
    }, []);

    if (loading) {
        return (
            <div className="space-y-2">
                {[1, 2, 3].map((i) => (
                    <div className="h-14 animate-pulse rounded-xl bg-white/[0.02]" key={i} />
                ))}
            </div>
        );
    }

    if (events.length === 0) {
        return (
            <p className="text-muted-foreground/30 px-4 py-6 text-center text-xs">
                No activity yet - spawn a world to get started
            </p>
        );
    }

    return (
        <div className="space-y-2">
            {events.map((event) => {
                const cfg = TYPE_CONFIG[event.type] ?? TYPE_CONFIG['world.session_ended'];
                const Icon = cfg.icon;
                const meta = [];
                if (event.cost_usd) {
                    meta.push(`$${event.cost_usd.toFixed(3)}`);
                }
                if (event.duration_ms) {
                    const s = Math.floor(event.duration_ms / 1000);
                    meta.push(s < 60 ? `${s}s` : `${Math.floor(s / 60)}m`);
                }

                return (
                    <div
                        className="group flex items-center gap-4 rounded-xl px-4 py-3 transition-all hover:bg-white/[0.03]"
                        key={event.id}
                    >
                        <div
                            className={`h-8 w-8 rounded-lg ${cfg.bg} flex shrink-0 items-center justify-center`}
                        >
                            <Icon className={cfg.color} size={14} />
                        </div>
                        <div className="min-w-0 flex-1">
                            <p className="text-foreground/70 truncate text-xs">{event.phrase}</p>
                            {meta.length > 0 && (
                                <p className="text-muted-foreground/25 mt-0.5 font-mono text-[10px]">
                                    {meta.join(' · ')}
                                </p>
                            )}
                        </div>
                        <span className="text-muted-foreground/20 shrink-0 font-mono text-[10px]">
                            {timeAgo(event.timestamp)}
                        </span>
                    </div>
                );
            })}
        </div>
    );
}
