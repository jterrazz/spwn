'use client';

import type { AgentProfile, World } from '../../domain/model';
import { getWorldName } from '../../domain/model';
import { ROLE_BADGE } from '../../styles/status-colors';
import {
    KeyValue,
    MetricGrid,
    SectionHeader,
    SectionLabel,
    Separator,
    StatusDot,
    SubLabel,
} from '../ds';

interface AgentDiagnosticsProps {
    agent: string;
    profile: AgentProfile;
    world: null | World;
    files: number;
}

export function agentMetrics(profile: AgentProfile, files: number) {
    return [
        { label: 'Files', value: files },
        { label: 'Journal', value: profile.journal.length },
        { label: 'Skills', value: profile.skills.length },
        { label: 'Traits', value: profile.traits.length },
    ];
}

function Status({ status }: { status: string }) {
    return (
        <div className="flex items-center justify-between">
            <SubLabel>Status</SubLabel>
            <div className="flex items-center gap-1.5">
                <StatusDot status={status} />
                <span className="text-foreground/80 font-mono text-xs font-medium capitalize">
                    {status}
                </span>
            </div>
        </div>
    );
}

function Identity({ agent, profile, world }: Omit<AgentDiagnosticsProps, 'files'>) {
    const deployed = world?.agents.find((member) => member.name === agent);
    const role = deployed?.role ?? profile.role;
    return (
        <div>
            <SectionLabel>Identity</SectionLabel>
            <div className="space-y-2">
                <KeyValue label="Name" value={agent} />
                <div className="flex items-center justify-between">
                    <SubLabel>Role</SubLabel>
                    <span
                        className={`rounded border px-1.5 py-0.5 font-mono text-[9px] tracking-wider uppercase ${ROLE_BADGE[role] ?? ROLE_BADGE.default}`}
                    >
                        {role}
                    </span>
                </div>
                {deployed !== undefined && <Status status={deployed.status} />}
                <KeyValue label="Runtime" value={world?.runtime ?? profile.engine} />
                {profile.team !== undefined && profile.team !== '' && (
                    <KeyValue label="Team" value={profile.team} />
                )}
            </div>
        </div>
    );
}

function WorldFacts({ world }: { world: World }) {
    const workspaces = world.workspaces?.length ?? 0;
    return (
        <div>
            <SectionLabel>World</SectionLabel>
            <div className="space-y-2">
                <KeyValue label="Name" value={getWorldName(world)} />
                <Status status={world.status} />
                <KeyValue label="Agents" value={world.agents.length} />
                {workspaces > 0 && <KeyValue label="Workspaces" value={workspaces} />}
            </div>
        </div>
    );
}

/** The side column of an agent's page: who it is, what its mind holds, and the world it lives in. */
export function AgentDiagnostics({ agent, profile, world, files }: AgentDiagnosticsProps) {
    return (
        <div className="space-y-6 p-5">
            <SectionHeader>Diagnostics</SectionHeader>
            <Identity agent={agent} profile={profile} world={world} />
            <Separator />
            <div>
                <SectionLabel>Metrics</SectionLabel>
                <MetricGrid columns={2} items={agentMetrics(profile, files)} />
            </div>
            {world !== null && (
                <>
                    <Separator />
                    <WorldFacts world={world} />
                </>
            )}
            <Separator />
        </div>
    );
}
