'use client';

import { useId, useState } from 'react';

import { apiAction, apiPut, encPath } from '../../api/client';
import type { Team } from '../../domain/model';
import { focusOnMount } from '../focus-on-mount';
import { Modal, ModalActions } from '../modal';

const FIELD =
    'text-foreground/80 placeholder:text-muted-foreground/30 w-full rounded-lg border border-white/[0.08] bg-white/[0.04] px-3 py-2.5 text-sm transition-colors focus:border-white/[0.16] focus:outline-none disabled:opacity-50';

interface NewAgentDialogProps {
    teams: Team[];
    onCreated: (name: string) => void;
    onClose: () => void;
}

function useCreateAgent(onCreated: (name: string) => void) {
    const [creating, setCreating] = useState(false);
    const [error, setError] = useState('');
    const create = async (name: string, team: string) => {
        setCreating(true);
        setError('');
        const result = await apiAction('/api/agents', { name });
        if (result.ok && team !== '') {
            await apiPut(`/api/agents/${encPath(name)}/identity`, { content: team, field: 'team' });
        }
        setCreating(false);
        if (result.ok) {
            onCreated(name);
        } else {
            setError(result.error ?? 'Failed to create agent');
        }
    };
    return { create, creating, error };
}

function TeamSelect({
    team,
    teams,
    disabled,
    onTeam,
}: {
    team: string;
    teams: Team[];
    disabled: boolean;
    onTeam: (team: string) => void;
}) {
    const id = useId();
    return (
        <>
            <label
                className="text-muted-foreground/40 mt-4 mb-1.5 block text-[10px] tracking-widest uppercase"
                htmlFor={id}
            >
                Team{' '}
                <span className="text-muted-foreground/25 tracking-normal normal-case">
                    (optional)
                </span>
            </label>
            <select
                className={FIELD}
                disabled={disabled}
                id={id}
                onChange={(event) => {
                    onTeam(event.target.value);
                }}
                value={team}
            >
                <option value="">No team</option>
                {teams.map((entry) => (
                    <option key={entry.slug} value={entry.slug}>
                        {entry.name}
                    </option>
                ))}
            </select>
        </>
    );
}

/** Creates an agent in limbo, in a team or none. */
export function NewAgentDialog({ teams, onCreated, onClose }: NewAgentDialogProps) {
    const nameId = useId();
    const [name, setName] = useState('');
    const [team, setTeam] = useState('');
    const { create, creating, error } = useCreateAgent(onCreated);
    const submit = () => {
        if (name.trim() !== '') {
            void create(name.trim(), team);
        }
    };

    return (
        <Modal className="max-w-md p-6" closable={!creating} onClose={onClose}>
            <h3 className="font-heading text-foreground/90 mb-1 text-lg">New Agent</h3>
            <label className="text-muted-foreground/50 mb-5 block text-sm" htmlFor={nameId}>
                Creates a new agent identity in limbo. Deploy it to a world when ready.
            </label>
            <input
                className={`${FIELD} font-mono`}
                disabled={creating}
                id={nameId}
                onChange={(event) => {
                    setName(event.target.value);
                }}
                onKeyDown={(event) => {
                    if (event.key === 'Enter') {
                        submit();
                    } else if (event.key === 'Escape') {
                        onClose();
                    }
                }}
                placeholder="e.g. atlas, morpheus, neo..."
                ref={focusOnMount}
                value={name}
            />
            <TeamSelect disabled={creating} onTeam={setTeam} team={team} teams={teams} />
            {error !== '' && <p className="mt-3 text-xs text-red-400/80">{error}</p>}
            <ModalActions
                busy={creating}
                busyLabel="Creating..."
                label="Create"
                onCancel={onClose}
                onConfirm={submit}
            />
        </Modal>
    );
}
