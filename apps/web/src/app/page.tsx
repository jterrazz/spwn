'use client';

import { useRouter } from 'next/navigation';
import { useCallback, useEffect, useState } from 'react';

import { apiDelete, apiGet } from '../api/client';
import { useRefetch } from '../components/app-shell';
import { Page } from '../components/page';
import { PageHeader } from '../components/page-header';
import { Skeleton } from '../components/ui/skeleton';
import { EmptyWorlds } from '../components/worlds/empty-worlds';
import { HeaderStats } from '../components/worlds/header-stats';
import { QuickStart } from '../components/worlds/quick-start';
import { SpawnWorldDialog } from '../components/worlds/spawn-world-dialog';
import { WorldCarousel } from '../components/worlds/world-carousel';
import type { World } from '../domain/model';
import { inBackground } from '../events/in-background';
import { useBackgroundTask } from '../hooks/use-background-task';
import { useKeyboardShortcuts } from '../hooks/use-keyboard-shortcuts';
import { usePageTitle } from '../hooks/use-page-title';

const SLEEPING = new Set(['idle', 'waiting']);

function useWorldsAndAgents() {
    const [worlds, setWorlds] = useState<null | World[]>(null);
    const [agentCount, setAgentCount] = useState<null | number>(null);

    const fetchWorlds = useCallback(async () => {
        try {
            setWorlds(await apiGet<World[]>('/api/worlds'));
        } catch {
            setWorlds([]);
        }
    }, []);
    const fetchAgents = useCallback(async () => {
        try {
            const agents = await apiGet<unknown[]>('/api/agents');
            setAgentCount(agents.length);
        } catch {
            setAgentCount(0);
        }
    }, []);

    useBackgroundTask(fetchWorlds, 5000);
    useBackgroundTask(fetchAgents);
    return { agentCount, fetchAgents, fetchWorlds, worlds };
}

function nextIndex(key: string, selected: null | number, count: number): null | number | undefined {
    if (key === 'ArrowRight' || key === 'd') {
        return selected === null ? 0 : (selected + 1) % count;
    }
    if (key === 'ArrowLeft' || key === 'a') {
        return selected === null ? count - 1 : (selected - 1 + count) % count;
    }
    return key === 'Escape' && selected !== null ? null : undefined;
}

function useWorldKeys(
    worlds: World[],
    selected: null | number,
    setSelected: (index: null | number) => void,
    paused: boolean,
) {
    const router = useRouter();
    useEffect(() => {
        const onKey = (event: KeyboardEvent) => {
            if (paused || worlds.length === 0) {
                return;
            }
            const world = selected === null ? undefined : worlds[selected];
            if (event.key === 'Enter' && world !== undefined) {
                router.push(`/world/${world.id}`);
                return;
            }
            const next = nextIndex(event.key, selected, worlds.length);
            if (next !== undefined) {
                setSelected(next);
            }
        };
        globalThis.addEventListener('keydown', onKey);
        return () => {
            globalThis.removeEventListener('keydown', onKey);
        };
    }, [worlds, selected, router, paused, setSelected]);
}

function agentsCounted(worlds: World[], states: Set<string>): number {
    return worlds.reduce(
        (count, world) => count + world.agents.filter((agent) => states.has(agent.status)).length,
        0,
    );
}

function LoadingWorlds() {
    return (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
            {[1, 2, 3].map((index) => (
                <Skeleton className="h-32 rounded-lg" key={index} />
            ))}
        </div>
    );
}

function useSpawnDialog() {
    const [spawning, setSpawning] = useState(false);
    const openSpawn = useCallback(() => {
        setSpawning(true);
    }, []);
    const closeSpawn = useCallback(() => {
        setSpawning(false);
    }, []);
    useKeyboardShortcuts({ onEscape: closeSpawn, onSpawnWorld: openSpawn });
    return { closeSpawn, openSpawn, spawning };
}

interface WorldsBodyProps {
    worlds: null | World[];
    agentCount: null | number;
    selected: null | number;
    onSelect: (index: null | number) => void;
    onShutdown: (world: World) => void;
    onSpawn: () => void;
    onRefresh: () => void;
}

function WorldsBody({
    worlds,
    agentCount,
    selected,
    onSelect,
    onShutdown,
    onSpawn,
    onRefresh,
}: WorldsBodyProps) {
    const router = useRouter();
    if (worlds === null) {
        return <LoadingWorlds />;
    }
    if (worlds.length > 0) {
        return (
            <WorldCarousel
                onEnter={(world) => {
                    router.push(`/world/${world.id}`);
                }}
                onSelect={onSelect}
                onShutdown={onShutdown}
                onSpawn={onSpawn}
                selected={selected}
                worlds={worlds}
            />
        );
    }
    return agentCount === 0 ? (
        <QuickStart onComplete={onRefresh} />
    ) : (
        <EmptyWorlds agentCount={agentCount ?? 0} onRefetch={onRefresh} onSpawn={onSpawn} />
    );
}

export default function UniverseMapPage() {
    const refetchSidebar = useRefetch();
    const { agentCount, fetchAgents, fetchWorlds, worlds } = useWorldsAndAgents();
    const [selected, setSelected] = useState<null | number>(null);
    const { closeSpawn, openSpawn, spawning } = useSpawnDialog();
    const known = worlds ?? [];
    usePageTitle('Worlds');
    useWorldKeys(known, selected, setSelected, spawning);

    const refresh = () => {
        void fetchWorlds();
        void fetchAgents();
        refetchSidebar();
    };
    const shutdown = async (world: World) => {
        await apiDelete(`/api/worlds/${world.id}`);
        setSelected(null);
        refresh();
    };
    return (
        <Page className="flex h-full flex-col">
            <PageHeader
                actions={
                    <HeaderStats
                        idleAgents={agentsCounted(known, SLEEPING)}
                        onSpawn={openSpawn}
                        runningAgents={agentsCounted(known, new Set(['running']))}
                        worldsCount={known.length}
                    />
                }
                description="Isolated environments where your agents live and work."
                title="Worlds"
            />
            <WorldsBody
                agentCount={agentCount}
                onRefresh={refresh}
                onSelect={setSelected}
                onShutdown={inBackground(shutdown)}
                onSpawn={openSpawn}
                selected={selected}
                worlds={worlds}
            />
            {spawning && <SpawnWorldDialog onClose={closeSpawn} onComplete={refresh} />}
        </Page>
    );
}
