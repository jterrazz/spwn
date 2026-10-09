'use client';

import { IconRocket, IconX } from '@tabler/icons-react';
import { useRouter } from 'next/navigation';
import { useCallback, useState } from 'react';

import { apiGet } from '../../api/client';
import { errorMessageOf, SPAWN_PROGRESS, spawnWorld } from '../../api/worlds';
import { inBackground } from '../../events/in-background';
import { useBackgroundTask } from '../../hooks/use-background-task';
import { useProgressMessages } from '../../hooks/use-progress-messages';
import { ProgressShimmer } from '../progress-shimmer';
import { AgentPicker } from './agent-picker';
import { ConfigFields, NameField, SpawnPreview, workspaceDraft } from './spawn-world-parts';
import type { Mount, WorkspaceDraft } from './spawn-world-parts';
import { WorkspaceEditor } from './workspace-editor';

interface SpawnWorldDialogProps {
    onClose: () => void;
    onComplete: () => void;
}

function scratchSuffix(): string {
    return Math.random().toString(36).slice(2, 6);
}

function toggled(names: Set<string>, name: string): Set<string> {
    const next = new Set(names);
    if (next.has(name)) {
        next.delete(name);
    } else {
        next.add(name);
    }
    return next;
}

function mountsOf(workspaces: WorkspaceDraft[], defaultPath: string): Mount[] {
    return workspaces
        .map((draft, index) => ({
            name: draft.name.trim() || (workspaces.length === 1 ? 'default' : `w${index}`),
            path: draft.path.trim() || (index === 0 ? defaultPath : ''),
            readonly: draft.readonly,
        }))
        .filter((mount) => mount.path !== '');
}

function useAgents() {
    const [agents, setAgents] = useState<string[]>([]);
    const load = useCallback(async () => {
        try {
            const listed = await apiGet<{ name: string }[]>('/api/agents');
            setAgents(listed.map((agent) => agent.name));
        } catch {
            setAgents([]);
        }
    }, []);
    useBackgroundTask(load);
    return { agents, setAgents };
}

function useSpawnForm() {
    const [worldName, setWorldName] = useState('');
    const [selected, setSelected] = useState<Set<string>>(new Set());
    const [workspaces, setWorkspaces] = useState<WorkspaceDraft[]>(() => [workspaceDraft()]);
    const [config, setConfig] = useState('default');
    const [role, setRole] = useState('worker');
    const [first] = selected;
    const defaultPath = `/tmp/spwn-${first ?? 'workspace'}-…`;

    const request = (suffix: string) => ({
        agents: [...selected].map((name) => ({ name, role })),
        config,
        name: worldName.trim(),
        role,
        workspaces: mountsOf(workspaces, `/tmp/spwn-${first ?? 'workspace'}-${suffix}`),
    });

    return {
        config,
        defaultPath,
        request,
        role,
        handleConfig: setConfig,
        handleRole: setRole,
        handleWorkspaces: setWorkspaces,
        handleWorldName: setWorldName,
        selected,
        setSelected,
        workspaces,
        worldName,
    };
}

function DialogHeading({ spawning, onClose }: { spawning: boolean; onClose: () => void }) {
    return (
        <div className="flex items-center justify-between px-6 pt-6 pb-4">
            <div>
                <h2 className="font-heading text-foreground/90 text-lg">New World</h2>
                <p className="text-muted-foreground/40 mt-0.5 text-[11px]">
                    Create a new isolated world for your agent
                </p>
            </div>
            <button
                aria-label="Close"
                className="text-muted-foreground/40 hover:text-foreground/60 transition-colors disabled:cursor-not-allowed disabled:opacity-30"
                disabled={spawning}
                onClick={onClose}
                type="button"
            >
                <IconX size={18} />
            </button>
        </div>
    );
}

function SpawnButton({ spawning, onSpawn }: { spawning: boolean; onSpawn: () => void }) {
    const progress = useProgressMessages(spawning, SPAWN_PROGRESS);
    return (
        <>
            <button
                className={`text-foreground/70 hover:text-foreground/90 flex w-full items-center justify-center gap-2 rounded-xl border border-white/[0.08] bg-white/[0.06] py-3 text-sm font-medium transition-all hover:bg-white/[0.1] disabled:cursor-not-allowed disabled:opacity-30 ${spawning ? 'animate-pulse' : ''}`}
                disabled={spawning}
                onClick={onSpawn}
                type="button"
            >
                {spawning ? (
                    <div className="border-foreground/30 border-t-foreground/70 h-3.5 w-3.5 animate-spin rounded-full border-2" />
                ) : (
                    <IconRocket size={16} />
                )}
                {spawning ? 'Spawning...' : 'New World'}
            </button>
            <ProgressShimmer active={spawning} message={progress} />
        </>
    );
}

function SpawnForm({
    form,
    onError,
}: {
    form: ReturnType<typeof useSpawnForm>;
    onError: (error: string) => void;
}) {
    const { agents, setAgents } = useAgents();
    return (
        <>
            <NameField onChange={form.handleWorldName} value={form.worldName} />
            <AgentPicker
                available={agents}
                onClear={() => {
                    form.setSelected(new Set());
                }}
                onCreated={(name) => {
                    setAgents((previous) => [...previous, name]);
                    form.setSelected((previous) => new Set(previous).add(name));
                }}
                onError={onError}
                onToggle={(name) => {
                    form.setSelected((previous) => toggled(previous, name));
                }}
                selected={form.selected}
            />
            <WorkspaceEditor
                defaultPath={form.defaultPath}
                onChange={form.handleWorkspaces}
                workspaces={form.workspaces}
            />
            <ConfigFields
                config={form.config}
                onConfig={form.handleConfig}
                onRole={form.handleRole}
                role={form.role}
            />
            <SpawnPreview
                agents={[...form.selected]}
                config={form.config}
                mounts={mountsOf(form.workspaces, form.defaultPath)}
                role={form.role}
            />
        </>
    );
}

/** The dialog that spawns a world: its name, agents, workspaces, config and role, with the command it amounts to. */
export function SpawnWorldDialog({ onClose, onComplete }: SpawnWorldDialogProps) {
    const router = useRouter();
    const form = useSpawnForm();
    const [spawning, setSpawning] = useState(false);
    const [error, setError] = useState('');

    const spawn = async () => {
        setSpawning(true);
        setError('');
        try {
            const id = await spawnWorld(form.request(scratchSuffix()));
            onComplete();
            onClose();
            if (id !== '') {
                router.push(`/world/${id}`);
            }
        } catch (spawnError) {
            setError(errorMessageOf(spawnError, 'Failed to spawn world'));
            setSpawning(false);
        }
    };

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
            <button
                aria-label="Close"
                className="absolute inset-0 bg-black/40 backdrop-blur-sm"
                disabled={spawning}
                onClick={onClose}
                type="button"
            />
            <div className="bg-popover/95 relative z-10 mx-4 w-full max-w-md overflow-hidden rounded-2xl border border-white/[0.08] shadow-2xl backdrop-blur-md">
                {spawning && (
                    <div className="h-0.5 w-full overflow-hidden bg-white/[0.04]">
                        <div
                            className="h-full w-1/3 rounded-full bg-emerald-500/30"
                            style={{ animation: 'progressSlide 1.5s ease-in-out infinite' }}
                        />
                    </div>
                )}
                <DialogHeading onClose={onClose} spawning={spawning} />
                <div className="space-y-4 px-6 pb-6">
                    <SpawnForm form={form} onError={setError} />
                    {error !== '' && (
                        <div className="rounded-lg border border-red-500/20 bg-red-500/10 px-3 py-2 font-mono text-xs text-red-400">
                            {error}
                        </div>
                    )}
                    <SpawnButton onSpawn={inBackground(spawn)} spawning={spawning} />
                </div>
            </div>
        </div>
    );
}
