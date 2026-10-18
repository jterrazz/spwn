'use client';

import { IconActivity, IconMoonFilled, IconPlus, IconWorldFilled } from '@tabler/icons-react';
import { useState } from 'react';

import { ActionButton } from '../action-button';
import { GLASS_PILL_CLASS } from '../glass-pill';

const OPEN = 'cubic-bezier(0.16, 1, 0.3, 1)';

interface Stat {
    key: string;
    icon: React.ReactNode;
    value: number;
    label: string;
    tone: string;
    tracking: string;
    width: number;
}

function StatPill({ stat, expanded }: { stat: Stat; expanded: boolean }) {
    return (
        <button
            aria-label={`${stat.value} ${stat.label}`}
            className={`flex h-[30px] items-center gap-1.5 overflow-hidden rounded-full border border-transparent px-2 ${stat.tone}`}
            style={{ transition: `width 280ms ${OPEN}`, width: expanded ? stat.width : 52 }}
            type="button"
        >
            <span className="flex h-[18px] w-[18px] shrink-0 items-center justify-center self-center">
                <span className="block translate-y-[0.5px] leading-none">{stat.icon}</span>
            </span>
            <span className="flex items-baseline gap-1.5 self-center whitespace-nowrap">
                <span className="font-mono text-[12px] leading-none font-medium">{stat.value}</span>
                <span
                    className={`text-[9px] leading-none font-medium uppercase ${stat.tracking}`}
                    style={{
                        opacity: expanded ? 1 : 0,
                        transform: expanded ? 'translateX(0)' : 'translateX(-6px)',
                        transition: `opacity 180ms ease, transform 280ms ${OPEN}`,
                    }}
                >
                    {stat.label}
                </span>
            </span>
        </button>
    );
}

interface HeaderStatsProps {
    worldsCount: number;
    runningAgents: number;
    idleAgents: number;
    onSpawn: () => void;
}

/** The counters of the worlds page (worlds, agents alive, agents sleeping), each label unfolding on hover, and the button that spawns a world. */
export function HeaderStats({ worldsCount, runningAgents, idleAgents, onSpawn }: HeaderStatsProps) {
    const [expanded, setExpanded] = useState(false);
    const stats: Stat[] = [
        {
            icon: <IconWorldFilled size={15} />,
            key: 'worlds',
            label: 'worlds',
            tone: 'text-foreground/78',
            tracking: 'tracking-[0.13em]',
            value: worldsCount,
            width: 102,
        },
        {
            icon: <IconActivity size={14} stroke={2.2} />,
            key: 'running',
            label: 'alive',
            tone: 'text-emerald-100/95',
            tracking: 'tracking-[0.13em]',
            value: runningAgents,
            width: 92,
        },
        {
            icon: <IconMoonFilled size={14} />,
            key: 'idle',
            label: 'sleeping',
            tone: 'text-amber-100/95',
            tracking: 'tracking-[0.11em]',
            value: idleAgents,
            width: 112,
        },
    ];
    const open = () => {
        setExpanded(true);
    };
    const close = () => {
        setExpanded(false);
    };

    return (
        <div className="flex flex-wrap items-center justify-end gap-2 md:max-w-[620px]">
            <div
                className={`${GLASS_PILL_CLASS} flex h-[42px] flex-nowrap items-center justify-end gap-1 px-2.5 transition-all duration-300 ease-out`}
                onBlur={close}
                onFocus={open}
                onMouseEnter={open}
                onMouseLeave={close}
            >
                {stats.map((stat) => (
                    <StatPill expanded={expanded} key={stat.key} stat={stat} />
                ))}
            </div>
            <ActionButton
                compact
                icon={<IconPlus size={18} stroke={2.4} />}
                label="New World"
                onClick={onSpawn}
            />
        </div>
    );
}
