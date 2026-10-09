'use client';

import { Handle, Position } from '@xyflow/react';
import type { Node, NodeProps } from '@xyflow/react';

import type { OrganizationRole } from '../../domain/model';
import type { RoleNodeData } from './flow';

interface Palette {
    border: string;
    text: string;
    bg: string;
    glow: string;
}

const PALETTES: Partial<Record<number, Palette>> = {
    0: {
        bg: 'bg-amber-500/10',
        border: 'border-amber-500/30',
        glow: 'shadow-[0_0_20px_rgba(245,158,11,0.08)]',
        text: 'text-amber-300',
    },
    1: {
        bg: 'bg-purple-500/10',
        border: 'border-purple-500/25',
        glow: 'shadow-[0_0_16px_rgba(168,85,247,0.06)]',
        text: 'text-purple-300',
    },
    2: { bg: 'bg-blue-500/10', border: 'border-blue-500/20', glow: '', text: 'text-blue-300' },
};
const DEEP: Palette = {
    bg: 'bg-white/[0.04]',
    border: 'border-white/[0.1]',
    glow: '',
    text: 'text-foreground/60',
};
const HANDLE = '!h-1.5 !w-1.5 !border-0 !bg-white/[0.15]';
const CHIP =
    'text-muted-foreground/40 rounded border border-white/[0.05] bg-white/[0.03] px-1.5 py-0.5 font-mono text-[8px]';
const LINK = 'text-muted-foreground/25 font-mono text-[8px]';

function Badges({ role, palette }: { role: OrganizationRole; palette: Palette }) {
    return (
        <div className="mt-2 flex items-center justify-center gap-1.5">
            <span
                className={`rounded-full border px-2 py-0.5 font-mono text-[9px] ${palette.bg} ${palette.text} ${palette.border}`}
            >
                Level {role.level}
            </span>
            {(role.max_per_world ?? 0) > 0 && (
                <span className="text-muted-foreground/40 rounded-full border border-white/[0.06] bg-white/[0.03] px-2 py-0.5 font-mono text-[9px]">
                    max {role.max_per_world}
                </span>
            )}
        </div>
    );
}

function Links({ role }: { role: OrganizationRole }) {
    const reportsTo = role.reports_to ?? '';
    const commands = role.can_command ?? [];
    return (
        <div className="mt-2.5 space-y-0.5">
            {reportsTo !== '' && (
                <p className={LINK}>
                    reports to <span className="text-muted-foreground/40">{reportsTo}</span>
                </p>
            )}
            {commands.length > 0 && (
                <p className={LINK}>
                    commands <span className="text-muted-foreground/40">{commands.join(', ')}</span>
                </p>
            )}
        </div>
    );
}

/** One role on the chart: its name, level, cap, permissions and links, coloured by depth. */
export function RoleNode({ data }: NodeProps<Node<RoleNodeData>>) {
    const { role, isFirst, isLast } = data;
    const palette = PALETTES[role.level] ?? DEEP;
    return (
        <div
            className={`min-w-[220px] rounded-xl border bg-[#0a0a0c]/90 px-6 py-4 text-center backdrop-blur-sm transition-shadow ${palette.border} ${palette.glow}`}
        >
            {!isFirst && <Handle className={HANDLE} position={Position.Top} type="target" />}
            {!isLast && <Handle className={HANDLE} position={Position.Bottom} type="source" />}
            <p
                className={`font-mono text-sm font-bold tracking-[0.06em] uppercase ${palette.text}`}
            >
                {role.name}
            </p>
            <Badges palette={palette} role={role} />
            {(role.permissions ?? []).length > 0 && (
                <div className="mt-3 flex flex-wrap items-center justify-center gap-1">
                    {(role.permissions ?? []).map((permission) => (
                        <span className={CHIP} key={permission}>
                            {permission}
                        </span>
                    ))}
                </div>
            )}
            <Links role={role} />
        </div>
    );
}
