'use client';

import { IconCamera, IconFileText, IconPencil, IconRestore, IconTrash } from '@tabler/icons-react';
import { useRouter } from 'next/navigation';

import type { Agent, World } from '../../domain/model';
import { ROLE_BADGE } from '../../styles/status-colors';
import { ActionButton } from '../action-button';
import { DataTable } from '../data-table';
import { SectionHeader, Separator, StatusDot } from '../ds';
import { Skeleton } from '../ui/skeleton';
import { EmptyAgents } from './empty-agents';
import type { Panel } from './world-side-panel';

export type Prompt = 'destroy' | 'rename' | null;

export function timeAgo(iso: string): string {
    if (iso === '') {
        return '-';
    }
    const minutes = Math.floor((Date.now() - new Date(iso).getTime()) / 60_000);
    if (minutes < 60) {
        return `${minutes}m ago`;
    }
    const hours = Math.floor(minutes / 60);
    return hours < 24 ? `${hours}h ago` : `${Math.floor(hours / 24)}d ago`;
}

const AGENT_COLUMNS = [
    {
        key: 'name',
        label: 'Name',
        render: (agent: Agent) => (
            <span className="text-foreground/85 truncate font-mono text-[13px]">{agent.name}</span>
        ),
        width: '1fr',
    },
    {
        key: 'role',
        label: 'Role',
        render: (agent: Agent) => (
            <span
                className={`rounded border px-1.5 py-0.5 font-mono text-[9px] tracking-wider uppercase ${ROLE_BADGE[agent.role] ?? ROLE_BADGE.default}`}
            >
                {agent.role}
            </span>
        ),
        width: '80px',
    },
    {
        key: 'status',
        label: 'Status',
        render: (agent: Agent) => (
            <span className="flex items-center gap-1.5">
                <StatusDot status={agent.status} />
                <span className="text-muted-foreground/50 font-mono text-[11px] capitalize">
                    {agent.status}
                </span>
            </span>
        ),
        width: '100px',
    },
];

export function WorldLoading() {
    return (
        <div className="space-y-8 p-8">
            <div className="flex items-center gap-4">
                <Skeleton className="h-2.5 w-2.5 rounded-full" />
                <div className="space-y-2">
                    <Skeleton className="h-7 w-32" />
                    <Skeleton className="h-3 w-48" />
                </div>
            </div>
            <div className="flex gap-10">
                {[1, 2, 3, 4].map((index) => (
                    <div key={index}>
                        <Skeleton className="mb-2 h-3 w-14" />
                        <Skeleton className="h-7 w-10" />
                    </div>
                ))}
            </div>
            <Skeleton className="h-16 rounded-xl" />
        </div>
    );
}

export function WorldAgents({ world, onDeployed }: { world: World; onDeployed: () => void }) {
    const router = useRouter();
    return (
        <div>
            <div className="mb-3 flex items-center justify-between">
                <SectionHeader className="mb-0">Agents</SectionHeader>
                {world.agents.length > 0 && (
                    <button
                        className="text-muted-foreground/35 hover:text-foreground/70 font-mono text-[10px] transition-colors"
                        onClick={() => {
                            router.push('/agents');
                        }}
                        type="button"
                    >
                        + Deploy
                    </button>
                )}
            </div>
            {world.agents.length === 0 ? (
                <EmptyAgents onDeployed={onDeployed} worldId={world.id} />
            ) : (
                <DataTable<Agent>
                    columns={AGENT_COLUMNS}
                    emptyText="No agents deployed."
                    rowHref={(agent) =>
                        `/agents/${encodeURIComponent(agent.name)}?world=${world.id}`
                    }
                    rowKey={(agent) => agent.name}
                    rows={world.agents}
                />
            )}
        </div>
    );
}

export function Elements({ elements }: { elements: string[] }) {
    if (elements.length === 0) {
        return null;
    }
    return (
        <>
            <div>
                <SectionHeader>Elements</SectionHeader>
                <div className="flex flex-wrap gap-1.5">
                    {elements.map((element) => (
                        <span
                            className="text-foreground/60 border border-white/[0.06] bg-white/[0.04] px-2.5 py-1 font-mono text-[11px]"
                            key={element}
                        >
                            {element}
                        </span>
                    ))}
                </div>
            </div>
            <Separator />
        </>
    );
}

interface WorldActionsProps {
    busy: boolean;
    onSnapshot: () => void;
    onPanel: (panel: Panel) => void;
    onPrompt: (prompt: Prompt) => void;
}

export function WorldActions({ busy, onSnapshot, onPanel, onPrompt }: WorldActionsProps) {
    const verbs = [
        { handleClick: onSnapshot, icon: <IconCamera size={16} stroke={2.2} />, label: 'Snapshot' },
        {
            handleClick: () => {
                onPanel('logs');
            },
            icon: <IconFileText size={16} stroke={2.2} />,
            label: 'Logs',
        },
        {
            handleClick: () => {
                onPanel('snapshots');
            },
            icon: <IconRestore size={16} stroke={2.2} />,
            label: 'Snapshots',
        },
        {
            handleClick: () => {
                onPrompt('rename');
            },
            icon: <IconPencil size={16} stroke={2.2} />,
            label: 'Rename',
        },
    ];
    return (
        <>
            {verbs.map((verb) => (
                <ActionButton
                    compact
                    disabled={busy && verb.label === 'Snapshot'}
                    icon={verb.icon}
                    key={verb.label}
                    label={verb.label}
                    onClick={verb.handleClick}
                />
            ))}
            <ActionButton
                compact
                danger
                icon={<IconTrash size={16} stroke={2.2} />}
                label="Destroy"
                onClick={() => {
                    onPrompt('destroy');
                }}
            />
        </>
    );
}
