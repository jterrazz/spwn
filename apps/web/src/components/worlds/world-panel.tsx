'use client';

import { IconArrowRight, IconX } from '@tabler/icons-react';

import type { Agent, World } from '../../domain/model';
import { getWorldName } from '../../domain/model';
import { ItemList, MetricGrid, ProgressBar, SectionLabel, Separator, StatusDot } from '../ds';
import { WorldPlanet } from '../world-planet';

const ALIVE = new Set(['idle', 'running', 'waiting']);

function uptimeOf(createdAt: string): string {
    if (createdAt === '') {
        return '-';
    }
    const minutes = Math.floor((Date.now() - new Date(createdAt).getTime()) / 60_000);
    if (minutes < 60) {
        return `${minutes}m`;
    }
    const hours = Math.floor(minutes / 60);
    return hours < 24 ? `${hours}h` : `${Math.floor(hours / 24)}d`;
}

function aliveShare(agents: Agent[]): number {
    const alive = agents.filter((agent) => ALIVE.has(agent.status)).length;
    return agents.length === 0 ? 0 : Math.round((alive / agents.length) * 100);
}

interface WorldPanelProps {
    world: World;
    panelRef: React.Ref<HTMLDivElement>;
    onClose: () => void;
    onEnter: () => void;
    onShutdown: () => void;
}

function PanelTitle({ world, onClose }: Pick<WorldPanelProps, 'onClose' | 'world'>) {
    return (
        <div className="flex items-center justify-between">
            <div className="flex min-w-0 items-center gap-3">
                <WorldPlanet size="md" world={world} />
                <div className="min-w-0">
                    <h3 className="text-foreground/95 truncate font-mono text-sm font-bold">
                        {getWorldName(world)}
                    </h3>
                    <p className="text-muted-foreground/35 truncate font-mono text-[10px]">
                        {world.config} ·{' '}
                        <StatusDot className="inline-block align-middle" status={world.status} />{' '}
                        {world.status}
                    </p>
                </div>
            </div>
            <button
                aria-label="Close"
                className="text-muted-foreground/30 hover:text-foreground/70 flex h-7 w-7 shrink-0 items-center justify-center rounded-full transition-colors"
                onClick={onClose}
                type="button"
            >
                <IconX size={14} />
            </button>
        </div>
    );
}

function PanelLists({ world }: { world: World }) {
    const workspaces = world.workspaces ?? [];
    return (
        <>
            {world.agents.length > 0 && (
                <ProgressBar label="Alive" value={aliveShare(world.agents)} />
            )}
            {world.agents.length > 0 && (
                <div>
                    <SectionLabel>Agents</SectionLabel>
                    <ItemList
                        items={world.agents.map((agent) => ({
                            detail: agent.status,
                            href: `/agents/${encodeURIComponent(agent.name)}?world=${world.id}`,
                            name: agent.name,
                        }))}
                    />
                </div>
            )}
            {workspaces.length > 0 && (
                <div>
                    <SectionLabel>Workspaces</SectionLabel>
                    <ItemList
                        items={workspaces.map((workspace) => ({
                            detail:
                                workspace.readonly === true
                                    ? `${workspace.path} (ro)`
                                    : workspace.path,
                            name: workspace.name,
                        }))}
                    />
                </div>
            )}
        </>
    );
}

function PanelActions({
    world,
    onEnter,
    onShutdown,
}: Pick<WorldPanelProps, 'onEnter' | 'onShutdown' | 'world'>) {
    const running = world.status === 'running' || world.status === 'idle';
    return (
        <div className="flex items-center gap-2">
            <button
                className="text-foreground/70 hover:text-foreground/95 flex h-9 flex-1 items-center justify-center gap-2 rounded-full border border-white/[0.06] bg-white/[0.06] font-mono text-xs font-medium transition-all hover:border-white/[0.12] hover:bg-white/[0.1]"
                onClick={onEnter}
                type="button"
            >
                Enter World
                <IconArrowRight size={13} />
            </button>
            {running && (
                <button
                    className="text-muted-foreground/30 h-9 rounded-full border border-transparent px-3.5 font-mono text-[11px] transition-all hover:border-red-500/15 hover:bg-red-500/[0.06] hover:text-red-400"
                    onClick={onShutdown}
                    type="button"
                >
                    Shutdown
                </button>
            )}
        </div>
    );
}

/** The panel beside the carousel for the world it has selected: its state, its agents and workspaces, and the way in. */
export function WorldPanel({ world, panelRef, onClose, onEnter, onShutdown }: WorldPanelProps) {
    return (
        <div className="pointer-events-none absolute inset-y-0 right-6 z-10 flex w-[340px] items-center pb-24 md:right-8">
            <div
                className="border-foreground/[0.08] bg-foreground/[0.04] animate-in fade-in slide-in-from-right-12 pointer-events-auto w-full overflow-hidden rounded-2xl border shadow-[inset_0_1px_0_rgba(255,255,255,0.08),0_1px_2px_rgba(0,0,0,0.04)] backdrop-blur-md duration-500 ease-out dark:border-white/[0.1] dark:bg-white/[0.05] dark:shadow-[inset_0_1px_0_rgba(255,255,255,0.05),0_1px_2px_rgba(0,0,0,0.18)]"
                ref={panelRef}
            >
                <div className="space-y-5 p-5">
                    <PanelTitle onClose={onClose} world={world} />
                    <MetricGrid
                        columns={3}
                        items={[
                            { label: 'Uptime', value: uptimeOf(world.created_at) },
                            { label: 'Agents', value: world.agents.length },
                            { label: 'Workspaces', value: world.workspaces?.length ?? 0 },
                        ]}
                    />
                    <Separator />
                    <PanelLists world={world} />
                    <PanelActions onEnter={onEnter} onShutdown={onShutdown} world={world} />
                </div>
            </div>
        </div>
    );
}
