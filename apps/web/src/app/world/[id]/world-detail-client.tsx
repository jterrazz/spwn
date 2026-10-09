'use client';

import { useParams, useRouter } from 'next/navigation';
import { useState } from 'react';

import { apiAction, apiDelete } from '../../../api/client';
import { useRefetch } from '../../../components/app-shell';
import { MetricGrid, Separator } from '../../../components/ds';
import { PageHeader } from '../../../components/page-header';
import { useToast } from '../../../components/toast-provider';
import { WorldPlanet } from '../../../components/world-planet';
import { DestroyConfirm } from '../../../components/world/destroy-confirm';
import { RenameDialog } from '../../../components/world/rename-dialog';
import { useWorld } from '../../../components/world/use-world';
import {
    Elements,
    timeAgo,
    WorldActions,
    WorldAgents,
    WorldLoading,
} from '../../../components/world/world-parts';
import type { Prompt } from '../../../components/world/world-parts';
import { WorldSidePanel } from '../../../components/world/world-side-panel';
import type { Panel } from '../../../components/world/world-side-panel';
import { getWorkspaceSummary, getWorldName } from '../../../domain/model';
import type { World } from '../../../domain/model';
import { inBackground } from '../../../events/in-background';
import { usePageTitle } from '../../../hooks/use-page-title';

type WorldPage = ReturnType<typeof useWorldPage>;

function useWorldPage() {
    const { id: worldId } = useParams<{ id: string }>();
    const router = useRouter();
    const { toast } = useToast();
    const refetchSidebar = useRefetch();
    const { fetchWorld, loading, world } = useWorld(worldId);
    const [busy, setBusy] = useState<'destroy' | 'snapshot' | null>(null);
    usePageTitle(world === null ? null : getWorldName(world));

    const report = (message: string) => {
        toast(message, message.toLowerCase().startsWith('error') ? 'error' : 'success');
    };
    const refresh = () => {
        void fetchWorld();
        refetchSidebar();
    };
    const snapshot = async () => {
        setBusy('snapshot');
        const result = await apiAction(`/api/worlds/${worldId}/snapshot`);
        setBusy(null);
        report(result.ok ? 'Snapshot saved!' : `Error: ${result.error ?? 'Unknown error'}`);
        refresh();
    };
    const destroy = async () => {
        setBusy('destroy');
        try {
            await apiDelete(`/api/worlds/${worldId}`);
            report('World destroyed');
            refetchSidebar();
            router.push('/');
        } catch {
            report('Error: Failed to destroy world');
        }
        setBusy(null);
    };
    return {
        busy,
        destroy,
        handleRefresh: refresh,
        handleReport: report,
        loading,
        snapshot,
        world,
        worldId,
    };
}

function WorldPrompts({
    page,
    world,
    prompt,
    onClose,
}: {
    page: WorldPage;
    world: World;
    prompt: Prompt;
    onClose: () => void;
}) {
    return (
        <>
            {prompt === 'rename' && (
                <RenameDialog
                    onClose={() => {
                        onClose();
                    }}
                    onFailed={page.handleReport}
                    onRenamed={() => {
                        page.handleRefresh();
                        onClose();
                    }}
                    world={world}
                />
            )}
            {prompt === 'destroy' && (
                <DestroyConfirm
                    destroying={page.busy === 'destroy'}
                    onCancel={() => {
                        onClose();
                    }}
                    onDestroy={inBackground(page.destroy)}
                    worldId={world.id}
                />
            )}
        </>
    );
}

function WorldMain({
    page,
    world,
    panel,
    onPanel,
}: {
    page: WorldPage;
    world: World;
    panel: null | Panel;
    onPanel: (panel: null | Panel) => void;
}) {
    const [prompt, setPrompt] = useState<Prompt>(null);
    return (
        <div className="flex-1 space-y-6 overflow-y-auto px-6 pt-6 pb-12 md:space-y-8 md:px-8 md:pt-8">
            <PageHeader
                actions={
                    <WorldActions
                        busy={page.busy !== null}
                        onPanel={(next) => {
                            onPanel(panel === next ? null : next);
                        }}
                        onPrompt={setPrompt}
                        onSnapshot={inBackground(page.snapshot)}
                    />
                }
                description={[
                    world.config,
                    world.created_at === '' ? '' : timeAgo(world.created_at),
                    getWorkspaceSummary(world),
                ]
                    .filter((part) => part !== '')
                    .join(' · ')}
                leading={<WorldPlanet size="lg" world={world} />}
                title={getWorldName(world)}
            />
            <WorldPrompts
                onClose={() => {
                    setPrompt(null);
                }}
                page={page}
                prompt={prompt}
                world={world}
            />
            <MetricGrid
                className="w-fit gap-x-10"
                columns={4}
                items={[
                    { label: 'Status', value: world.status },
                    { label: 'Agents', value: world.agents.length },
                    { label: 'Config', value: world.config },
                    { label: 'Uptime', value: timeAgo(world.created_at) },
                ]}
            />
            <Separator />
            <Elements elements={world.manifest?.elements ?? []} />
            <WorldAgents onDeployed={page.handleRefresh} world={world} />
        </div>
    );
}

export default function WorldDashboard() {
    const page = useWorldPage();
    const [panel, setPanel] = useState<null | Panel>(null);
    const { world } = page;

    if (page.loading) {
        return <WorldLoading />;
    }
    if (world === null) {
        return (
            <div className="p-8">
                <p className="text-muted-foreground/50">World not found</p>
            </div>
        );
    }

    return (
        <div className="flex h-[calc(100vh-1px)] overflow-hidden">
            <WorldMain onPanel={setPanel} page={page} panel={panel} world={world} />
            {panel !== null && (
                <WorldSidePanel
                    onClose={() => {
                        setPanel(null);
                    }}
                    panel={panel}
                    worldId={world.id}
                />
            )}
        </div>
    );
}
