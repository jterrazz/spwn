'use client';

import { IconX } from '@tabler/icons-react';
import { useId } from 'react';

import { AVAILABLE_CONFIGS } from '../../domain/model';

export const LABEL = 'text-muted-foreground/40 text-[10px] tracking-widest uppercase';
const FIELD =
    'text-foreground/80 placeholder:text-muted-foreground/25 w-full rounded-lg border border-white/[0.08] bg-white/[0.03] px-3 py-2.5 text-sm transition-colors focus:border-white/[0.15] focus:outline-none';
const SMALL_FIELD =
    'text-foreground/80 placeholder:text-muted-foreground/25 rounded-lg border border-white/[0.08] bg-white/[0.03] px-2.5 py-2 font-mono text-xs transition-colors focus:border-white/[0.15] focus:outline-none';
const ROLES = [
    { label: 'Chief', value: 'chief' },
    { label: 'Manager', value: 'manager' },
    { label: 'Worker', value: 'worker' },
];

export interface WorkspaceDraft {
    id: string;
    name: string;
    path: string;
    readonly: boolean;
}

let drafts = 0;

export function workspaceDraft(name = 'default'): WorkspaceDraft {
    drafts += 1;
    return { id: `ws-${drafts}`, name, path: '', readonly: false };
}

export function NameField({
    value,
    onChange,
}: {
    value: string;
    onChange: (value: string) => void;
}) {
    const id = useId();
    return (
        <div>
            <label className={`${LABEL} mb-1.5 block`} htmlFor={id}>
                Name{' '}
                <span className="text-muted-foreground/25 tracking-normal normal-case">
                    (optional)
                </span>
            </label>
            <input
                className={FIELD}
                id={id}
                onChange={(event) => {
                    onChange(event.target.value);
                }}
                placeholder="My Project"
                value={value}
            />
        </div>
    );
}

interface WorkspaceRowProps {
    draft: WorkspaceDraft;
    placeholder: string;
    onChange: (draft: WorkspaceDraft) => void;
    onRemove: () => void;
}

export function WorkspaceRow({ draft, placeholder, onChange, onRemove }: WorkspaceRowProps) {
    const tone = draft.readonly
        ? 'border border-amber-500/25 bg-amber-500/15 text-amber-300'
        : 'text-muted-foreground/40 hover:text-foreground/70 border border-white/[0.08] bg-white/[0.03]';
    return (
        <div className="flex items-center gap-1.5">
            <input
                aria-label="Workspace name"
                className={`${SMALL_FIELD} w-24 shrink-0`}
                onChange={(event) => {
                    onChange({ ...draft, name: event.target.value });
                }}
                placeholder="name"
                value={draft.name}
            />
            <input
                aria-label="Workspace path"
                className={`${SMALL_FIELD} min-w-0 flex-1`}
                onChange={(event) => {
                    onChange({ ...draft, path: event.target.value });
                }}
                placeholder={placeholder}
                value={draft.path}
            />
            <button
                className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-lg font-mono text-[10px] transition-colors ${tone}`}
                onClick={() => {
                    onChange({ ...draft, readonly: !draft.readonly });
                }}
                title={draft.readonly ? 'Read-only' : 'Read-write'}
                type="button"
            >
                ro
            </button>
            <button
                aria-label="Remove the workspace"
                className="text-muted-foreground/30 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg transition-colors hover:bg-red-500/10 hover:text-red-400"
                onClick={onRemove}
                type="button"
            >
                <IconX size={14} />
            </button>
        </div>
    );
}

interface ChoiceProps {
    label: string;
    value: string;
    options: { label: string; value: string }[];
    onChange: (value: string) => void;
}

function Choice({ label, value, options, onChange }: ChoiceProps) {
    const id = useId();
    return (
        <div>
            <label className={`${LABEL} mb-1.5 block`} htmlFor={id}>
                {label}
            </label>
            <select
                className={FIELD}
                id={id}
                onChange={(event) => {
                    onChange(event.target.value);
                }}
                value={value}
            >
                {options.map((option) => (
                    <option key={option.value} value={option.value}>
                        {option.label}
                    </option>
                ))}
            </select>
        </div>
    );
}

interface ConfigFieldsProps {
    config: string;
    role: string;
    onConfig: (config: string) => void;
    onRole: (role: string) => void;
}

export function ConfigFields({ config, role, onConfig, onRole }: ConfigFieldsProps) {
    return (
        <div className="grid grid-cols-2 gap-3">
            <Choice
                label="Config"
                onChange={onConfig}
                options={AVAILABLE_CONFIGS.map((name) => ({ label: name, value: name }))}
                value={config}
            />
            <Choice label="Agent Role" onChange={onRole} options={ROLES} value={role} />
        </div>
    );
}

export interface Mount {
    name: string;
    path: string;
    readonly: boolean;
}

interface SpawnPreviewProps {
    agents: string[];
    mounts: Mount[];
    role: string;
    config: string;
}

function flagsOf(agents: string[], mounted: Mount[]): { agentFlags: string; mounts: string } {
    const mounts = mounted
        .map((draft) => `-w ${draft.name}=${draft.path}${draft.readonly ? ':ro' : ''}`)
        .join(' ');
    return {
        agentFlags: agents.length > 0 ? agents.map((name) => `-a ${name}`).join(' ') : '--no-agent',
        mounts: mounts === '' ? ' (ephemeral)' : ` ${mounts}`,
    };
}

export function SpawnPreview({ agents, mounts: mounted, role, config }: SpawnPreviewProps) {
    const { agentFlags, mounts } = flagsOf(agents, mounted);
    return (
        <div className="space-y-2 rounded-lg border border-white/[0.05] bg-white/[0.02] px-3 py-3">
            <p className="text-muted-foreground/30 mb-1 text-[10px] tracking-widest uppercase">
                Preview
            </p>
            <div className="text-muted-foreground/35 font-mono text-[11px] break-all">
                spwn up {agentFlags} --role {role} --config {config}
                {mounts}
            </div>
            <div className="text-muted-foreground/25 space-y-0.5 text-[10px]">
                <p>→ Creates isolated Docker container</p>
                {agents.length === 0 ? (
                    <p>→ Empty world (no agents deployed)</p>
                ) : (
                    <p>
                        → Deploys {agents.length} agent{agents.length === 1 ? '' : 's'}:{' '}
                        <span className="font-mono">{agents.join(', ')}</span>
                    </p>
                )}
                {mounted.length === 0 && <p>→ No host workspace (uses image&apos;s /workspace)</p>}
                {mounted.map((draft) => (
                    <p key={draft.name}>
                        → {draft.name}: <span className="font-mono">{draft.path}</span>
                        {draft.readonly ? ' (read-only)' : ''}
                    </p>
                ))}
            </div>
        </div>
    );
}
