import { ROLE_BADGE } from '../../styles/status-colors';
import { StatusDot } from '../ds';
import type { RosterAgent } from './roster';

export const AGENT_COLUMNS = [
    {
        key: 'name',
        label: 'Name',
        render: (agent: RosterAgent) => (
            <span className="text-foreground/85 truncate font-mono text-[13px]">{agent.name}</span>
        ),
        width: '1fr',
    },
    {
        key: 'role',
        label: 'Role',
        render: (agent: RosterAgent) => (
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
        render: (agent: RosterAgent) => (
            <span className="flex items-center gap-1.5">
                <StatusDot status={agent.status === 'limbo' ? 'stopped' : agent.status} />
                <span className="text-muted-foreground/50 font-mono text-[11px] capitalize">
                    {agent.status}
                </span>
            </span>
        ),
        width: '100px',
    },
    {
        key: 'world',
        label: 'World',
        render: (agent: RosterAgent) =>
            agent.worldName === undefined || agent.worldName === '' ? (
                <span className="text-muted-foreground/25 font-mono text-[11px]">-</span>
            ) : (
                <span className="text-foreground/60 truncate font-mono text-[11px]">
                    {agent.worldName}
                </span>
            ),
        width: '120px',
    },
];
