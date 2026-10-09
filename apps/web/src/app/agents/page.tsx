'use client';

import { IconPlus, IconUser, IconUsers } from '@tabler/icons-react';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

import { ActionButton } from '../../components/action-button';
import { NewAgentDialog } from '../../components/agents/new-agent-dialog';
import { byTeam, countsOf, filtered } from '../../components/agents/roster';
import type { StatusFilter } from '../../components/agents/roster';
import { RosterGroups } from '../../components/agents/roster-groups';
import { StatusFilters } from '../../components/agents/status-filter';
import { TeamDialog } from '../../components/agents/team-dialog';
import { useRoster } from '../../components/agents/use-roster';
import { useRefetch } from '../../components/app-shell';
import { ExpandingSearch } from '../../components/expanding-search';
import { Page } from '../../components/page';
import { PageHeader } from '../../components/page-header';
import { Skeleton } from '../../components/ui/skeleton';
import type { Team } from '../../domain/model';
import { usePageTitle } from '../../hooks/use-page-title';

type Dialog = null | { kind: 'agent' } | { kind: 'team'; team: null | Team };

function Loading() {
    return (
        <div className="space-y-2">
            {[1, 2, 3].map((index) => (
                <Skeleton className="h-14 w-full rounded-xl" key={index} />
            ))}
        </div>
    );
}

function Empty({ searching }: { searching: boolean }) {
    return (
        <div className="flex flex-col items-center justify-center py-16 text-center">
            <div className="mb-4 flex h-14 w-14 items-center justify-center rounded-2xl border border-white/[0.06] bg-white/[0.03]">
                <IconUser className="text-muted-foreground/30" size={24} />
            </div>
            <p className="text-muted-foreground/50 text-sm">
                {searching
                    ? 'No agents match your search'
                    : 'No agents yet. Create one to get started.'}
            </p>
        </div>
    );
}

function AgentsDialog({
    dialog,
    teams,
    onClose,
    onSaved,
}: {
    dialog: Dialog;
    teams: Team[];
    onClose: () => void;
    onSaved: () => Promise<void>;
}) {
    const router = useRouter();
    const refetchSidebar = useRefetch();
    return (
        <>
            {dialog?.kind === 'agent' && (
                <NewAgentDialog
                    onClose={onClose}
                    onCreated={(name) => {
                        onClose();
                        refetchSidebar();
                        void onSaved();
                        router.push(`/agents/${encodeURIComponent(name)}`);
                    }}
                    teams={teams}
                />
            )}
            {dialog?.kind === 'team' && (
                <TeamDialog
                    onClose={onClose}
                    onSaved={() => {
                        onClose();
                        void onSaved();
                    }}
                    team={dialog.team}
                />
            )}
        </>
    );
}

export default function AgentsPage() {
    const { agents, fetchAll, loading, teams } = useRoster();
    const [filter, setFilter] = useState<StatusFilter>('all');
    const [query, setQuery] = useState('');
    const [dialog, setDialog] = useState<Dialog>(null);
    usePageTitle('Agents');
    const shown = filtered(agents, filter, query);
    const close = () => {
        setDialog(null);
    };

    return (
        <Page>
            <PageHeader
                actions={
                    <>
                        <ExpandingSearch
                            onChange={setQuery}
                            placeholder="Search agents…"
                            value={query}
                        />
                        <ActionButton
                            compact
                            icon={<IconUsers size={16} stroke={2.2} />}
                            label="New Team"
                            onClick={() => {
                                setDialog({ kind: 'team', team: null });
                            }}
                        />
                        <ActionButton
                            compact
                            icon={<IconPlus size={18} stroke={2.4} />}
                            label="New Agent"
                            onClick={() => {
                                setDialog({ kind: 'agent' });
                            }}
                        />
                    </>
                }
                description="Persistent identities that remember across sessions and worlds."
                title="Agents"
            />
            <StatusFilters counts={countsOf(agents)} filter={filter} onFilter={setFilter} />
            {loading && <Loading />}
            {!loading && shown.length === 0 && <Empty searching={query !== ''} />}
            {!loading && shown.length > 0 && (
                <RosterGroups
                    groups={byTeam(shown, teams)}
                    onEditTeam={(team) => {
                        setDialog({ kind: 'team', team });
                    }}
                />
            )}
            <AgentsDialog dialog={dialog} onClose={close} onSaved={fetchAll} teams={teams} />
        </Page>
    );
}
