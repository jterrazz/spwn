'use client';

import {
    IconBrain,
    IconPlus,
    IconRocket,
    IconUserFilled,
    IconWorldFilled,
} from '@tabler/icons-react';
import { useRouter } from 'next/navigation';
import { useCallback, useEffect, useState } from 'react';

import { apiGet } from '../api/client';
import { getWorldName } from '../domain/model';
import type { World } from '../domain/model';
import {
    Command,
    CommandDialog,
    CommandEmpty,
    CommandGroup,
    CommandInput,
    CommandItem,
    CommandList,
    CommandSeparator,
} from './ui/command';

const ICON = 'text-muted-foreground/50';
const HINT = 'text-muted-foreground/30 -mr-6 flex-1 text-right font-mono text-[10px]';

async function optional<Value>(promise: Promise<Value>, fallback: Value): Promise<Value> {
    try {
        return await promise;
    } catch {
        return fallback;
    }
}

function usePaletteShortcut(toggle: () => void): void {
    useEffect(() => {
        const handleKeyDown = (event: KeyboardEvent) => {
            if ((event.metaKey || event.ctrlKey) && event.key === 'k') {
                event.preventDefault();
                toggle();
            }
        };
        globalThis.addEventListener('keydown', handleKeyDown);
        return () => {
            globalThis.removeEventListener('keydown', handleKeyDown);
        };
    }, [toggle]);
}

function useCatalogue(open: boolean) {
    const [worlds, setWorlds] = useState<World[]>([]);
    const [agents, setAgents] = useState<{ name: string }[]>([]);
    useEffect(() => {
        if (!open) {
            return;
        }
        const load = async () => {
            const [listedWorlds, listedAgents] = await Promise.all([
                optional(apiGet<World[]>('/api/worlds'), []),
                optional(apiGet<{ name: string }[]>('/api/agents'), []),
            ]);
            setWorlds(listedWorlds);
            setAgents(listedAgents);
        };
        void load();
    }, [open]);
    return { agents, worlds };
}

function openSpawnDialog(): void {
    globalThis.dispatchEvent(
        new KeyboardEvent('keydown', { bubbles: true, key: 'n', metaKey: true }),
    );
}

function AgentItems({
    agents,
    worlds,
    navigate,
}: {
    agents: { name: string }[];
    worlds: World[];
    navigate: (path: string) => void;
}) {
    return (
        <>
            {agents.map((agent) => {
                const world = worlds.find((candidate) =>
                    candidate.agents.some((member) => member.name === agent.name),
                );
                const path = `/agents/${encodeURIComponent(agent.name)}`;
                return (
                    <CommandItem
                        key={agent.name}
                        onSelect={() => {
                            navigate(world === undefined ? path : `${path}?world=${world.id}`);
                        }}
                    >
                        <IconUserFilled className={ICON} size={14} />
                        <span>{agent.name}</span>
                        <span className={HINT}>
                            {world === undefined ? 'limbo' : `in ${getWorldName(world)}`}
                        </span>
                    </CommandItem>
                );
            })}
        </>
    );
}

function NavigationGroup({ navigate }: { navigate: (path: string) => void }) {
    return (
        <CommandGroup heading="Navigation">
            <CommandItem
                onSelect={() => {
                    navigate('/');
                }}
            >
                <IconWorldFilled className={ICON} size={14} />
                <span>Go to Dashboard</span>
            </CommandItem>
            <CommandItem
                onSelect={() => {
                    navigate('/architect');
                }}
            >
                <IconBrain className={ICON} size={14} />
                <span>Go to Architect</span>
            </CommandItem>
        </CommandGroup>
    );
}

function WorldItems({ worlds, navigate }: { worlds: World[]; navigate: (path: string) => void }) {
    if (worlds.length === 0) {
        return null;
    }
    return (
        <CommandGroup heading="Worlds">
            {worlds.map((world) => (
                <CommandItem
                    key={world.id}
                    onSelect={() => {
                        navigate(`/world/${world.id}`);
                    }}
                >
                    <IconWorldFilled className={ICON} size={14} />
                    <span>{getWorldName(world)}</span>
                    <span className={HINT}>
                        {world.agents.length} agents · {world.status}
                    </span>
                </CommandItem>
            ))}
        </CommandGroup>
    );
}

function Actions({ navigate }: { navigate: (path: string) => void }) {
    return (
        <CommandGroup heading="Actions">
            <CommandItem
                onSelect={() => {
                    navigate('/');
                    setTimeout(openSpawnDialog, 100);
                }}
            >
                <IconRocket className={ICON} size={14} />
                <span>Spawn World</span>
                <span className={HINT}>⌘N</span>
            </CommandItem>
            <CommandItem
                onSelect={() => {
                    navigate('/agents');
                }}
            >
                <IconPlus className={ICON} size={14} />
                <span>Create Agent</span>
            </CommandItem>
        </CommandGroup>
    );
}

/** The ⌘K palette: jump to a world or an agent, open a page, or start a world or an agent. */
export function CommandPalette() {
    const [open, setOpen] = useState(false);
    const router = useRouter();
    const { agents, worlds } = useCatalogue(open);
    const toggle = useCallback(() => {
        setOpen((previous) => !previous);
    }, []);
    usePaletteShortcut(toggle);
    const navigate = useCallback(
        (path: string) => {
            setOpen(false);
            router.push(path);
        },
        [router],
    );

    return (
        <CommandDialog onOpenChange={setOpen} open={open}>
            <Command className="bg-popover/95 rounded-xl border border-white/[0.08] backdrop-blur-md">
                <CommandInput placeholder="Search worlds, agents, commands..." />
                <CommandList>
                    <CommandEmpty>No results found.</CommandEmpty>
                    <WorldItems navigate={navigate} worlds={worlds} />
                    {agents.length > 0 && (
                        <>
                            <CommandSeparator />
                            <CommandGroup heading="Agents">
                                <AgentItems agents={agents} navigate={navigate} worlds={worlds} />
                            </CommandGroup>
                        </>
                    )}
                    <CommandSeparator />
                    <NavigationGroup navigate={navigate} />
                    <CommandSeparator />
                    <Actions navigate={navigate} />
                </CommandList>
            </Command>
        </CommandDialog>
    );
}
