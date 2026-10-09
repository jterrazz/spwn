'use client';

import { encPath } from '../../api/client';
import type { AgentProfile, Team } from '../../domain/model';
import { inBackground } from '../../events/in-background';
import { ItemList, MetricGrid, SectionHeader, Separator, SubLabel } from '../ds';
import { InlineEdit } from '../inline-edit';
import { InlineTagsEdit } from '../inline-tags-edit';
import { agentMetrics } from './agent-diagnostics';

interface ProfileTabProps {
    agent: string;
    profile: AgentProfile;
    teams: Team[];
    journalFiles: string[];
    files: number;
    onSave: (field: string, content: string) => Promise<boolean>;
}

function worldTitleOf(worldId: string): string {
    const [, second] = worldId.split('-');
    return second === undefined ? worldId : second.charAt(0).toUpperCase() + second.slice(1);
}

function DeploymentHistory({ agent, journalFiles }: { agent: string; journalFiles: string[] }) {
    const worldIds = journalFiles
        .filter((file) => file.endsWith('.json'))
        .map((file) => file.replace('.json', ''));
    if (worldIds.length === 0) {
        return null;
    }
    return (
        <>
            <Separator />
            <div>
                <SectionHeader>Deployment History</SectionHeader>
                <ItemList
                    items={worldIds.map((worldId) => ({
                        detail: worldId,
                        href: `/agents/${encPath(agent)}?world=${worldId}`,
                        name: worldTitleOf(worldId),
                    }))}
                />
            </div>
        </>
    );
}

function Identity({ profile, onSave }: Pick<ProfileTabProps, 'onSave' | 'profile'>) {
    return (
        <div>
            <SectionHeader>Identity</SectionHeader>
            <div className="mb-4">
                <SubLabel className="mb-1.5">Purpose</SubLabel>
                <InlineEdit
                    className="text-foreground/75 text-sm leading-relaxed"
                    multiline
                    onSave={async (value) => await onSave('purpose', value)}
                    placeholder="Define this agent's purpose..."
                    value={profile.purpose}
                />
            </div>
            <div className="mb-4">
                <SubLabel className="mb-1.5">Profile</SubLabel>
                <InlineEdit
                    className="text-foreground/60 text-sm leading-relaxed italic"
                    multiline
                    onSave={async (value) => await onSave('profile', value)}
                    placeholder="Describe the agent's profile..."
                    value={profile.profile}
                />
            </div>
            <div>
                <SubLabel className="mb-1.5">Traits</SubLabel>
                <InlineTagsEdit
                    color="bg-white/[0.06] text-foreground/60 border-white/[0.08]"
                    onSave={async (tags) =>
                        await onSave('traits', tags.map((tag) => `- ${tag}`).join('\n'))
                    }
                    tags={profile.traits}
                />
            </div>
        </div>
    );
}

function SkillsAndJournal({ profile }: { profile: AgentProfile }) {
    return (
        <>
            {profile.skills.length > 0 && (
                <>
                    <Separator />
                    <div>
                        <SectionHeader>Skills</SectionHeader>
                        <div className="flex flex-wrap gap-1.5">
                            {profile.skills.map((skill) => (
                                <span
                                    className="text-foreground/60 border border-white/[0.06] bg-white/[0.04] px-2.5 py-1 font-mono text-[11px]"
                                    key={skill}
                                >
                                    {skill}
                                </span>
                            ))}
                        </div>
                    </div>
                </>
            )}
            {profile.journal.length > 0 && (
                <>
                    <Separator />
                    <div>
                        <SectionHeader>Journal</SectionHeader>
                        <div className="space-y-3">
                            {profile.journal.map((entry) => (
                                <div key={entry.date}>
                                    <SubLabel className="mb-1">{entry.date}</SubLabel>
                                    <p className="text-foreground/60 text-xs leading-relaxed">
                                        {entry.summary}
                                    </p>
                                </div>
                            ))}
                        </div>
                    </div>
                </>
            )}
        </>
    );
}

/** The agent's profile: its counts, its team, where it was deployed, its identity to edit, its skills and journal. */
export function ProfileTab({
    agent,
    profile,
    teams,
    journalFiles,
    files,
    onSave,
}: ProfileTabProps) {
    return (
        <>
            <MetricGrid
                className="gap-x-8 gap-y-4"
                columns={2}
                items={agentMetrics(profile, files)}
            />
            <Separator />
            <div className="flex items-center justify-between">
                <SubLabel>Team</SubLabel>
                <select
                    aria-label="Team"
                    className="text-foreground/80 cursor-pointer bg-transparent text-right font-mono text-sm focus:outline-none"
                    onChange={inBackground(async (event: React.ChangeEvent<HTMLSelectElement>) => {
                        await onSave('team', event.target.value);
                    })}
                    value={profile.team ?? ''}
                >
                    <option value="">-</option>
                    {teams.map((team) => (
                        <option key={team.slug} value={team.slug}>
                            {team.name}
                        </option>
                    ))}
                </select>
            </div>
            <DeploymentHistory agent={agent} journalFiles={journalFiles} />
            <Separator />
            <Identity onSave={onSave} profile={profile} />
            <SkillsAndJournal profile={profile} />
        </>
    );
}
