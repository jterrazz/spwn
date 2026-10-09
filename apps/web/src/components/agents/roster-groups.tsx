'use client';

import type { Team } from '../../domain/model';
import { DataTable } from '../data-table';
import { SectionLabel } from '../ds';
import { AGENT_COLUMNS } from './agent-columns';
import type { RosterAgent, TeamGroup } from './roster';

function addressOf(agent: RosterAgent): string {
    const path = `/agents/${encodeURIComponent(agent.name)}`;
    return agent.worldId === undefined || agent.worldId === ''
        ? path
        : `${path}?world=${agent.worldId}`;
}

function GroupHeading({
    team,
    count,
    onEditTeam,
}: {
    team: null | Team;
    count: number;
    onEditTeam: (team: Team) => void;
}) {
    if (team === null) {
        return (
            <>
                <SectionLabel className="text-muted-foreground/30 mb-0">No team</SectionLabel>
                <span className="text-muted-foreground/20 font-mono text-[10px]">{count}</span>
            </>
        );
    }
    return (
        <>
            <button
                className="underline-offset-2 transition-colors hover:underline"
                onClick={() => {
                    onEditTeam(team);
                }}
                style={
                    team.color === undefined || team.color === ''
                        ? undefined
                        : { color: team.color }
                }
                type="button"
            >
                <SectionLabel className="mb-0">{team.name}</SectionLabel>
            </button>
            <span className="text-muted-foreground/30 font-mono text-[10px]">{count}</span>
        </>
    );
}

/** The roster as one table per team, each team's name the button that edits it. */
export function RosterGroups({
    groups,
    onEditTeam,
}: {
    groups: TeamGroup[];
    onEditTeam: (team: Team) => void;
}) {
    return (
        <div className="space-y-8">
            {groups.map(({ team, agents }) => (
                <div key={team?.slug ?? 'solo'}>
                    <div className="mb-3 flex items-center gap-2">
                        <GroupHeading count={agents.length} onEditTeam={onEditTeam} team={team} />
                    </div>
                    <DataTable<RosterAgent>
                        columns={AGENT_COLUMNS}
                        rowHref={addressOf}
                        rowKey={(agent) => agent.name}
                        rows={agents}
                    />
                </div>
            ))}
        </div>
    );
}
