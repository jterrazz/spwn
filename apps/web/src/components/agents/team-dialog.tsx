'use client';

import { useId, useState } from 'react';

import { apiDelete, apiPost, apiPut } from '../../api/client';
import type { Team } from '../../domain/model';
import { inBackground } from '../../events/in-background';
import { focusOnMount } from '../focus-on-mount';
import { Modal, ModalActions } from '../modal';

const LABEL = 'text-muted-foreground/40 mb-1.5 block text-[10px] tracking-widest uppercase';
const FIELD =
    'text-foreground/80 placeholder:text-muted-foreground/30 w-full rounded-lg border border-white/[0.08] bg-white/[0.04] px-3 py-2.5 text-sm transition-colors focus:border-white/[0.16] focus:outline-none';

interface TeamDialogProps {
    team: null | Team;
    onSaved: () => void;
    onClose: () => void;
}

interface FieldProps {
    label: React.ReactNode;
    value: string;
    placeholder: string;
    mono?: boolean;
    focus?: boolean;
    onChange: (value: string) => void;
    onEnter?: () => void;
}

function Field({
    label,
    value,
    placeholder,
    mono = false,
    focus = false,
    onChange,
    onEnter,
}: FieldProps) {
    const id = useId();
    return (
        <div>
            <label className={LABEL} htmlFor={id}>
                {label}
            </label>
            <input
                className={mono ? `${FIELD} font-mono` : FIELD}
                id={id}
                onChange={(event) => {
                    onChange(event.target.value);
                }}
                onKeyDown={(event) => {
                    if (event.key === 'Enter') {
                        onEnter?.();
                    }
                }}
                placeholder={placeholder}
                ref={focus ? focusOnMount : undefined}
                value={value}
            />
        </div>
    );
}

function useTeamForm(team: null | Team, onSaved: () => void) {
    const [name, setName] = useState(team?.name ?? '');
    const [color, setColor] = useState(team?.color ?? '');
    const [description, setDescription] = useState(team?.description ?? '');
    const [saving, setSaving] = useState(false);

    const save = async () => {
        if (name.trim() === '') {
            return;
        }
        setSaving(true);
        const body = { color: color.trim(), description: description.trim(), name: name.trim() };
        try {
            await (team === null
                ? apiPost('/api/teams', body)
                : apiPut(`/api/teams/${team.slug}`, body));
            onSaved();
        } finally {
            setSaving(false);
        }
    };
    return {
        color,
        description,
        handleColor: setColor,
        handleDescription: setDescription,
        handleName: setName,
        name,
        save,
        saving,
    };
}

function TeamFields({ form }: { form: ReturnType<typeof useTeamForm> }) {
    return (
        <div className="space-y-3">
            <Field
                focus
                label="Name"
                onChange={form.handleName}
                onEnter={inBackground(form.save)}
                placeholder="e.g. Matrix Ops"
                value={form.name}
            />
            <Field
                label="Color"
                mono
                onChange={form.handleColor}
                placeholder="#8B5CF6 or purple"
                value={form.color}
            />
            <Field
                label={
                    <>
                        Description{' '}
                        <span className="text-muted-foreground/25 tracking-normal normal-case">
                            (optional)
                        </span>
                    </>
                }
                onChange={form.handleDescription}
                placeholder="What this team does…"
                value={form.description}
            />
        </div>
    );
}

/** Creates a team, or edits or deletes one. */
export function TeamDialog({ team, onSaved, onClose }: TeamDialogProps) {
    const form = useTeamForm(team, onSaved);
    const remove = async () => {
        if (team !== null) {
            await apiDelete(`/api/teams/${team.slug}`);
            onSaved();
        }
    };

    return (
        <Modal className="max-w-md p-6" closable={!form.saving} onClose={onClose}>
            <h3 className="font-heading text-foreground/90 mb-1 text-lg">
                {team === null ? 'New Team' : 'Edit Team'}
            </h3>
            <p className="text-muted-foreground/50 mb-5 text-sm">
                {team === null
                    ? 'Create a new team to group agents together.'
                    : `Editing ${team.name}`}
            </p>
            <TeamFields form={form} />
            {team !== null && (
                <button
                    className="mt-6 text-[11px] text-red-400/60 transition-colors hover:text-red-400"
                    onClick={inBackground(remove)}
                    type="button"
                >
                    Delete team
                </button>
            )}
            <ModalActions
                busy={form.saving}
                busyLabel="Saving…"
                label={team === null ? 'Create' : 'Save'}
                onCancel={onClose}
                onConfirm={inBackground(form.save)}
            />
        </Modal>
    );
}
