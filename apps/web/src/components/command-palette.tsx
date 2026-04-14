'use client';

import {
    IconBook2,
    IconBrain,
    IconPlus,
    IconRocket,
    IconUserFilled,
    IconWorldFilled,
} from '@tabler/icons-react';
import { useRouter } from 'next/navigation';
import { useCallback, useEffect, useState } from 'react';

import {
    Command,
    CommandDialog,
    CommandEmpty,
    CommandGroup,
    CommandInput,
    CommandItem,
    CommandList,
    CommandSeparator,
} from '@/components/ui/command';
import { apiGet } from '@/lib/api-client';
import type { World } from '@/lib/types';

interface AgentListItem {
    name: string;
    path: string;
    layers: Record<string, string[]>;
}

function extractName(id: string): string {
    const parts = id.split('-');
    return parts.length >= 2 ? parts[1].charAt(0).toUpperCase() + parts[1].slice(1) : id;
}

export function CommandPalette() {
    const [open, setOpen] = useState(false);
    const [worlds, setWorlds] = useState<World[]>([]);
    const [agents, setAgents] = useState<AgentListItem[]>([]);
    const router = useRouter();

    // Listen for Cmd+K
    useEffect(() => {
        const handleKeyDown = (e: KeyboardEvent) => {
            if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
                e.preventDefault();
                setOpen((prev) => !prev);
            }
        };
        globalThis.addEventListener('keydown', handleKeyDown);
        return () => globalThis.removeEventListener('keydown', handleKeyDown);
    }, []);

    // Fetch data when opening
    useEffect(() => {
        if (!open) {
            return;
        }
        Promise.all([
            apiGet<World[]>('/api/worlds').catch(() => [] as World[]),
            apiGet<AgentListItem[]>('/api/agents').catch(() => [] as AgentListItem[]),
        ]).then(([w, a]) => {
            setWorlds(w ?? []);
            setAgents(a ?? []);
        });
    }, [open]);

    const navigate = useCallback(
        (path: string) => {
            setOpen(false);
            router.push(path);
        },
        [router],
    );

    return (
        <CommandDialog onOpenChange={setOpen} open={open}>
            <Command className="rounded-xl border border-white/[0.08] bg-popover/95 backdrop-blur-md">
                <CommandInput placeholder="Search worlds, agents, commands..." />
                <CommandList>
                    <CommandEmpty>No results found.</CommandEmpty>

                    {/* Worlds */}
                    {worlds.length > 0 && (
                        <CommandGroup heading="Worlds">
                            {worlds.map((world) => (
                                <CommandItem
                                    key={world.id}
                                    onSelect={() => navigate(`/world/${world.id}`)}
                                >
                                    <IconWorldFilled
                                        className="text-muted-foreground/50"
                                        size={14}
                                    />
                                    <span>{extractName(world.id)}</span>
                                    <span className="flex-1 text-right -mr-6 text-[10px] font-mono text-muted-foreground/30">
                                        {world.agents.length} agents · {world.status}
                                    </span>
                                </CommandItem>
                            ))}
                        </CommandGroup>
                    )}

                    {/* Agents */}
                    {agents.length > 0 && (
                        <>
                            <CommandSeparator />
                            <CommandGroup heading="Agents">
                                {agents.map((agent) => {
                                    // Find which world this agent is in
                                    const agentWorld = worlds.find((w) =>
                                        w.agents.some((a) => a.name === agent.name),
                                    );
                                    const href = agentWorld
                                        ? `/agents/${encodeURIComponent(agent.name)}?world=${agentWorld.id}`
                                        : `/agents/${encodeURIComponent(agent.name)}`;
                                    return (
                                        <CommandItem
                                            key={agent.name}
                                            onSelect={() => navigate(href)}
                                        >
                                            <IconUserFilled
                                                className="text-muted-foreground/50"
                                                size={14}
                                            />
                                            <span>{agent.name}</span>
                                            {agentWorld && (
                                                <span className="flex-1 text-right -mr-6 text-[10px] font-mono text-muted-foreground/30">
                                                    in {extractName(agentWorld.id)}
                                                </span>
                                            )}
                                            {!agentWorld && (
                                                <span className="flex-1 text-right -mr-6 text-[10px] font-mono text-muted-foreground/20">
                                                    limbo
                                                </span>
                                            )}
                                        </CommandItem>
                                    );
                                })}
                            </CommandGroup>
                        </>
                    )}

                    {/* Navigation */}
                    <CommandSeparator />
                    <CommandGroup heading="Navigation">
                        <CommandItem onSelect={() => navigate('/')}>
                            <IconWorldFilled className="text-muted-foreground/50" size={14} />
                            <span>Go to Dashboard</span>
                        </CommandItem>
                        <CommandItem onSelect={() => navigate('/architect')}>
                            <IconBrain className="text-muted-foreground/50" size={14} />
                            <span>Go to Architect</span>
                        </CommandItem>
                        <CommandItem onSelect={() => navigate('/knowledge')}>
                            <IconBook2 className="text-muted-foreground/50" size={14} />
                            <span>Go to Knowledge</span>
                        </CommandItem>
                        {/* Marketplace - hidden until ready */}
                    </CommandGroup>

                    {/* Actions */}
                    <CommandSeparator />
                    <CommandGroup heading="Actions">
                        <CommandItem
                            onSelect={() => {
                                setOpen(false);
                                // Trigger spawn world dialog on home page
                                navigate('/');
                                setTimeout(() => {
                                    globalThis.dispatchEvent(
                                        new KeyboardEvent('keydown', {
                                            key: 'n',
                                            metaKey: true,
                                            bubbles: true,
                                        }),
                                    );
                                }, 100);
                            }}
                        >
                            <IconRocket className="text-muted-foreground/50" size={14} />
                            <span>Spawn World</span>
                            <span className="flex-1 text-right -mr-6 text-[10px] font-mono text-muted-foreground/20">
                                ⌘N
                            </span>
                        </CommandItem>
                        <CommandItem
                            onSelect={() => {
                                setOpen(false);
                                const name = prompt('Agent name:');
                                if (name?.trim()) {
                                    navigate(`/agents/${name.trim()}`);
                                }
                            }}
                        >
                            <IconPlus className="text-muted-foreground/50" size={14} />
                            <span>Create Agent</span>
                        </CommandItem>
                    </CommandGroup>
                </CommandList>
            </Command>
        </CommandDialog>
    );
}
