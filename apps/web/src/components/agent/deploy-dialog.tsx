'use client';

import { IconPlanet } from '@tabler/icons-react';
import { useId, useState } from 'react';

import { apiPost, encPath } from '../../api/client';
import { errorMessageOf, SPAWN_PROGRESS } from '../../api/worlds';
import { AVAILABLE_ROLES, getWorldName } from '../../domain/model';
import type { Organization, World } from '../../domain/model';
import { inBackground } from '../../events/in-background';
import { useProgressMessages } from '../../hooks/use-progress-messages';
import { Modal, WorkingStripe } from '../modal';
import { ProgressShimmer } from '../progress-shimmer';

const LABEL = 'text-muted-foreground/40 mb-2 block text-[10px] tracking-[0.15em] uppercase';
const DEPLOY_BUDGET_MS = 600_000;

interface DeployDialogProps {
    agent: string;
    worlds: World[];
    organizations: Organization[];
    onDeployed: (worldId: string) => void;
    onClose: () => void;
}

function rolesFor(world: undefined | World, organizations: Organization[]): string[] {
    const slug = world?.organization ?? '';
    const organization =
        slug === '' ? undefined : organizations.find((candidate) => candidate.slug === slug);
    return organization ? organization.roles.map((role) => role.name) : [...AVAILABLE_ROLES];
}

function worldTone(deployed: boolean, chosen: boolean): string {
    if (deployed) {
        return 'opacity-40 cursor-not-allowed';
    }
    return chosen ? 'bg-white/[0.06]' : 'hover:bg-white/[0.03]';
}

interface WorldChoicesProps {
    agent: string;
    worlds: World[];
    chosen: string;
    onChoose: (worldId: string) => void;
}

function WorldChoices({ agent, worlds, chosen, onChoose }: WorldChoicesProps) {
    if (worlds.length === 0) {
        return (
            <p className="text-muted-foreground/40 rounded-lg border border-white/[0.06] bg-white/[0.02] px-3 py-2.5 text-[11px]">
                No running worlds. Spawn one first from the Worlds page.
            </p>
        );
    }
    return (
        <div className="max-h-48 overflow-y-auto rounded-lg border border-white/[0.08] bg-white/[0.02]">
            {worlds.map((world) => {
                const deployed = world.agents.some((member) => member.name === agent);
                return (
                    <button
                        aria-label={getWorldName(world)}
                        className={`flex w-full items-center gap-3 px-3 py-2.5 text-left transition-colors ${worldTone(deployed, chosen === world.id)}`}
                        disabled={deployed}
                        key={world.id}
                        onClick={() => {
                            onChoose(world.id);
                        }}
                        type="button"
                    >
                        <span
                            className={`h-2 w-2 shrink-0 rounded-full ${chosen === world.id ? 'bg-emerald-400' : 'bg-white/[0.15]'}`}
                        />
                        <span className="min-w-0 flex-1">
                            <span className="text-foreground/80 block truncate text-sm">
                                {getWorldName(world)}
                            </span>
                            <span className="text-muted-foreground/35 font-mono text-[10px]">
                                {world.agents.length} agent{world.agents.length === 1 ? '' : 's'}
                                {deployed && ' · already deployed'}
                            </span>
                        </span>
                    </button>
                );
            })}
        </div>
    );
}

function DeployButtons({
    busy,
    ready,
    onCancel,
    onDeploy,
}: {
    busy: boolean;
    ready: boolean;
    onCancel: () => void;
    onDeploy: () => void;
}) {
    return (
        <div className="mt-6 flex justify-end gap-3">
            <button
                className="text-muted-foreground/60 hover:text-foreground/80 rounded-lg px-4 py-2 text-sm transition-colors hover:bg-white/[0.04] disabled:opacity-50"
                disabled={busy}
                onClick={onCancel}
                type="button"
            >
                Cancel
            </button>
            <button
                className="flex items-center gap-2 rounded-lg border border-emerald-500/20 bg-emerald-500/20 px-4 py-2 text-sm text-emerald-300 transition-colors hover:bg-emerald-500/30 disabled:opacity-50"
                disabled={busy || !ready}
                onClick={onDeploy}
                type="button"
            >
                {busy ? (
                    <div className="h-3 w-3 animate-spin rounded-full border-2 border-emerald-300/40 border-t-emerald-300" />
                ) : (
                    <IconPlanet size={14} />
                )}
                {busy ? 'Deploying…' : 'Deploy'}
            </button>
        </div>
    );
}

interface RolePickerProps {
    roles: string[];
    role: string;
    disabled: boolean;
    onRole: (role: string) => void;
}

function RolePicker({ roles, role, disabled, onRole }: RolePickerProps) {
    const id = useId();
    return (
        <>
            <label className={`${LABEL} mt-4`} htmlFor={id}>
                Role
            </label>
            <select
                className="text-foreground/80 w-full rounded-lg border border-white/[0.08] bg-white/[0.04] px-3 py-2.5 text-sm transition-colors focus:border-white/[0.16] focus:outline-none disabled:opacity-50"
                disabled={disabled}
                id={id}
                onChange={(event) => {
                    onRole(event.target.value);
                }}
                value={role}
            >
                {roles.map((name) => (
                    <option key={name} value={name}>
                        {name}
                    </option>
                ))}
            </select>
        </>
    );
}

function useDeploy(agent: string, onDeployed: (worldId: string) => void) {
    const [deploying, setDeploying] = useState(false);
    const [error, setError] = useState('');
    const progress = useProgressMessages(deploying, [
        { after: 0, text: 'Deploying agent...' },
        ...SPAWN_PROGRESS.slice(1),
    ]);

    const deploy = async (target: string, role: string) => {
        setDeploying(true);
        setError('');
        try {
            await apiPost(
                `/api/worlds/${target}/agents`,
                { name: agent, role },
                { signal: AbortSignal.timeout(DEPLOY_BUDGET_MS) },
            );
            onDeployed(target);
        } catch (deployError) {
            setError(errorMessageOf(deployError, 'Deploy failed'));
            setDeploying(false);
        }
    };

    return { deploy, deploying, error, progress };
}

/** Deploys the agent into a running world, in the role that world's organization offers. */
export function DeployDialog({
    agent,
    worlds,
    organizations,
    onDeployed,
    onClose,
}: DeployDialogProps) {
    const [target, setTarget] = useState('');
    const [role, setRole] = useState('worker');
    const { deploy, deploying, error, progress } = useDeploy(agent, onDeployed);
    return (
        <Modal closable={!deploying} onClose={onClose}>
            <WorkingStripe active={deploying} />
            <div className="p-6">
                <h3 className="font-heading text-foreground/90 mb-1 text-lg">Deploy to World</h3>
                <p className="text-muted-foreground/50 mb-5 text-sm">
                    Add <span className="text-foreground/70 font-mono">{agent}</span> to a running
                    world.
                </p>
                <span className={LABEL}>Select World</span>
                <WorldChoices agent={agent} chosen={target} onChoose={setTarget} worlds={worlds} />
                <RolePicker
                    disabled={deploying}
                    onRole={setRole}
                    role={role}
                    roles={rolesFor(
                        worlds.find((world) => world.id === target),
                        organizations,
                    )}
                />
                {error !== '' && <p className="mt-3 text-xs text-red-400/80">{error}</p>}
                <DeployButtons
                    busy={deploying}
                    onCancel={onClose}
                    onDeploy={inBackground(async () => {
                        await deploy(target, role);
                    })}
                    ready={target !== ''}
                />
                <ProgressShimmer active={deploying} message={progress} />
            </div>
        </Modal>
    );
}

export function deployedAddress(agent: string, worldId: string): string {
    return `/agents/${encPath(agent)}?world=${worldId}`;
}
