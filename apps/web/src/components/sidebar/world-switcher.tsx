'use client';

import { useRouter } from 'next/navigation';
import { useState } from 'react';

import { getWorldName } from '../../domain/model';
import type { World } from '../../domain/model';
import { WorldPlanet } from '../world-planet';

const INLINE_CHARACTER_BUDGET = 22;

function splitByBudget(others: World[], expanded: boolean): { inline: World[]; overflow: World[] } {
    if (expanded) {
        return { inline: others, overflow: [] };
    }
    const inline: World[] = [];
    const overflow: World[] = [];
    let budget = INLINE_CHARACTER_BUDGET;
    for (const world of others) {
        const cost = getWorldName(world).length + 3;
        if (budget - cost < 0 && inline.length > 0) {
            overflow.push(world);
        } else {
            inline.push(world);
            budget -= cost;
        }
    }
    return { inline, overflow };
}

function SelectedWorld({ world }: { world: World }) {
    const router = useRouter();
    return (
        <button
            className="bg-sidebar-accent/50 hover:bg-sidebar-accent flex w-full items-center gap-3 rounded-md px-2 py-2 text-left transition-colors"
            onClick={() => {
                router.push(`/world/${world.id}`);
            }}
            type="button"
        >
            <WorldPlanet size="md" world={world} />
            <span className="min-w-0 flex-1">
                <span className="text-foreground block truncate text-sm font-medium">
                    {getWorldName(world)}
                </span>
                <span className="text-muted-foreground/40 block text-[10px] tracking-widest uppercase">
                    {world.status} · {world.agents.length} agent
                    {world.agents.length === 1 ? '' : 's'}
                </span>
            </span>
        </button>
    );
}

function WorldPill({ world }: { world: World }) {
    const router = useRouter();
    return (
        <button
            className="group/switch text-muted-foreground/50 hover:text-foreground hover:bg-sidebar-accent/40 flex h-6 shrink-0 items-center gap-1.5 rounded-md pr-2 pl-1 text-xs transition-colors"
            onClick={() => {
                router.push(`/world/${world.id}`);
            }}
            type="button"
        >
            <WorldPlanet
                className="opacity-80 transition-opacity group-hover/switch:opacity-100"
                size="sm"
                world={world}
            />
            <span>{getWorldName(world)}</span>
        </button>
    );
}

function OtherWorlds({ others }: { others: World[] }) {
    const [expanded, setExpanded] = useState(false);
    if (others.length === 0) {
        return null;
    }
    const { inline, overflow } = splitByBudget(others, expanded);
    const toggle = (label: string) => (
        <button
            aria-label={expanded ? 'Collapse worlds' : 'Show all worlds'}
            className="text-muted-foreground/40 hover:text-foreground hover:bg-sidebar-accent/40 flex h-6 min-w-6 shrink-0 items-center justify-center rounded-md px-1.5 text-[11px] font-medium transition-colors"
            onClick={() => {
                setExpanded(!expanded);
            }}
            type="button"
        >
            {label}
        </button>
    );
    return (
        <div
            className={`${expanded ? 'flex flex-wrap' : 'flex items-center overflow-hidden'} -mx-1 gap-1 px-1`}
        >
            {inline.map((world) => (
                <WorldPill key={world.id} world={world} />
            ))}
            {!expanded && overflow.length > 0 && toggle(`+${overflow.length}`)}
            {expanded && toggle('Hide')}
        </div>
    );
}

/** The world the sidebar follows, and the others one click away. */
export function WorldSwitcher({
    worlds,
    selected,
}: {
    worlds: World[];
    selected: undefined | World;
}) {
    if (selected === undefined) {
        return <p className="text-muted-foreground/25 px-2 py-1.5 text-xs">No worlds running</p>;
    }
    return (
        <div className="space-y-1.5 px-1">
            <SelectedWorld world={selected} />
            <OtherWorlds others={worlds.filter((world) => world.id !== selected.id)} />
        </div>
    );
}
