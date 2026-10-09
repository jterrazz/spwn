'use client';

import { useParams, useRouter, useSearchParams } from 'next/navigation';
import { Suspense, useState } from 'react';

import { apiAction, apiDelete, encPath } from '../../../api/client';
import { AgentActions } from '../../../components/agent/agent-actions';
import { AgentChat } from '../../../components/agent/agent-chat';
import { AgentDiagnostics } from '../../../components/agent/agent-diagnostics';
import { AgentLoading, AgentMissing, TabBar } from '../../../components/agent/agent-page-parts';
import type { Tab } from '../../../components/agent/agent-page-parts';
import { DeployDialog, deployedAddress } from '../../../components/agent/deploy-dialog';
import { IdentityWizard } from '../../../components/agent/identity-wizard';
import { MindFiles } from '../../../components/agent/mind-files';
import { ProfileTab } from '../../../components/agent/profile-tab';
import {
    useAgentActions,
    useAgentProfile,
    useAgentSurroundings,
} from '../../../components/agent/use-agent';
import { useRefetch } from '../../../components/app-shell';
import { ConfirmDialog } from '../../../components/confirm-dialog';
import { PageHeader } from '../../../components/page-header';
import { inBackground } from '../../../events/in-background';
import { FeedbackLine, useFeedback } from '../../../hooks/use-feedback';
import { usePageTitle } from '../../../hooks/use-page-title';

type Dialog = 'delete' | 'deploy' | null;

export default function AgentProfilePageWrapper() {
    return (
        <Suspense>
            <AgentProfilePage />
        </Suspense>
    );
}

function useAgentPage() {
    const params = useParams<{ name: string }>();
    const agent = decodeURIComponent(params.name);
    const worldId = useSearchParams().get('world') ?? undefined;
    const { feedback, show } = useFeedback();
    const { fetchProfile, loading, mindTree, profile } = useAgentProfile(agent);
    const surroundings = useAgentSurroundings(agent, worldId);
    const actions = useAgentActions(agent, show, fetchProfile);
    usePageTitle(agent, 'Agent');
    return {
        actions,
        agent,
        feedback,
        fetchProfile,
        loading,
        mindTree,
        profile,
        show,
        surroundings,
        worldId,
    };
}

type AgentPage = ReturnType<typeof useAgentPage>;
type Loaded = AgentPage & { profile: NonNullable<AgentPage['profile']> };

function AgentDialogs({
    page,
    dialog,
    onClose,
}: {
    page: Loaded;
    dialog: Dialog;
    onClose: () => void;
}) {
    const router = useRouter();
    const refetchSidebar = useRefetch();
    const [deleting, setDeleting] = useState(false);
    const { agent, surroundings } = page;

    const remove = async () => {
        setDeleting(true);
        try {
            await apiDelete(`/api/agents/${encPath(agent)}`);
            router.push('/');
        } catch {
            page.show('Error: Failed to delete agent');
            setDeleting(false);
            onClose();
        }
    };

    if (dialog === 'deploy') {
        return (
            <DeployDialog
                agent={agent}
                onClose={onClose}
                onDeployed={(worldId) => {
                    refetchSidebar();
                    onClose();
                    router.push(deployedAddress(agent, worldId));
                }}
                organizations={surroundings.organizations}
                worlds={surroundings.worlds}
            />
        );
    }
    if (dialog === 'delete') {
        return (
            <ConfirmDialog
                busy={deleting}
                confirm="Delete Agent"
                confirming="Deleting..."
                onCancel={onClose}
                onConfirm={inBackground(remove)}
                title="Delete Agent"
            >
                Are you sure you want to delete{' '}
                <span className="text-foreground/70 font-mono">{agent}</span>? This will permanently
                remove all mind files, memories, and identity data.
            </ConfirmDialog>
        );
    }
    return null;
}

function TabContent({ page, tab, files }: { page: Loaded; tab: Tab; files: number }) {
    const { actions, agent, mindTree, profile } = page;
    if (tab === 'profile') {
        return (
            <ProfileTab
                agent={agent}
                files={files}
                journalFiles={mindTree.journal ?? []}
                onSave={actions.handleIdentity}
                profile={profile}
                teams={page.surroundings.teams}
            />
        );
    }
    return tab === 'chat' ? (
        <AgentChat agent={agent} worldId={page.worldId} />
    ) : (
        <MindFiles agent={agent} mindTree={mindTree} />
    );
}

function AgentMain({ page, files }: { page: Loaded; files: number }) {
    const { actions, agent, profile } = page;
    const [tab, setTab] = useState<Tab>('chat');
    const [dialog, setDialog] = useState<Dialog>(null);
    const [wizardDismissed, setWizardDismissed] = useState(false);

    return (
        <div className="min-w-0 flex-1 space-y-6 p-4 md:space-y-8 md:p-8">
            <PageHeader
                actions={
                    <AgentActions
                        act={actions.act}
                        busy={actions.running !== null}
                        onDelete={() => {
                            setDialog('delete');
                        }}
                        onDeploy={() => {
                            setDialog('deploy');
                        }}
                        profile={profile}
                        reload={page.fetchProfile}
                        report={page.show}
                    />
                }
                description={`${page.surroundings.world?.runtime ?? profile.engine} · ${profile.role}`}
                title={agent}
            />
            <AgentDialogs
                dialog={dialog}
                onClose={() => {
                    setDialog(null);
                }}
                page={page}
            />
            {profile.purpose === '' && !wizardDismissed && (
                <IdentityWizard
                    agent={agent}
                    onDismiss={() => {
                        setWizardDismissed(true);
                    }}
                    onSave={actions.handleIdentity}
                />
            )}
            <TabBar files={files} onTab={setTab} tab={tab} />
            <FeedbackLine feedback={page.feedback} />
            <TabContent files={files} page={page} tab={tab} />
        </div>
    );
}

function AgentProfilePage() {
    const page = useAgentPage();
    const { agent, profile } = page;

    if (page.loading) {
        return <AgentLoading />;
    }
    if (profile === null) {
        const create = async () => {
            const result = await apiAction('/api/agents', { name: agent });
            page.show(result.ok ? 'Agent created!' : `Error: ${result.error ?? 'Unknown error'}`);
            await page.fetchProfile();
        };
        return <AgentMissing agent={agent} onCreate={inBackground(create)} />;
    }

    const loaded = { ...page, profile };
    const files = Object.values(page.mindTree).reduce((count, layer) => count + layer.length, 0);
    return (
        <div className="flex h-[calc(100vh-1px)] overflow-hidden">
            <div className="min-w-0 flex-1 overflow-y-auto">
                <AgentMain files={files} page={loaded} />
            </div>
            <div className="border-border/30 hidden w-72 shrink-0 overflow-y-auto border-l lg:block">
                <AgentDiagnostics
                    agent={agent}
                    files={files}
                    profile={profile}
                    world={page.surroundings.world}
                />
            </div>
        </div>
    );
}
