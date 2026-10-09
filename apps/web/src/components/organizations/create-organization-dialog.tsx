'use client';

import { IconPlus } from '@tabler/icons-react';
import { useId, useState } from 'react';

import { apiAction } from '../../api/client';
import { focusOnMount } from '../focus-on-mount';
import { Modal, ModalActions } from '../modal';
import { emptyRoleDraft, organizationOf } from './role-draft';
import type { RoleDraft } from './role-draft';
import { RoleDraftCard } from './role-draft-card';

const FIELD =
    'text-foreground/80 placeholder:text-muted-foreground/25 w-full rounded-lg border border-white/[0.08] bg-white/[0.03] px-3 py-2.5 text-sm transition-colors focus:border-white/[0.15] focus:outline-none';
const LABEL = 'text-muted-foreground/40 mb-1.5 block text-[10px] tracking-widest uppercase';

function useRoleDrafts() {
    const [drafts, setDrafts] = useState<RoleDraft[]>(() => [emptyRoleDraft()]);
    const change = (id: string, patch: Partial<RoleDraft>) => {
        setDrafts((previous) =>
            previous.map((draft) => (draft.id === id ? { ...draft, ...patch } : draft)),
        );
    };
    const remove = (id: string) => {
        setDrafts((previous) => previous.filter((draft) => draft.id !== id));
    };
    const handleAdd = () => {
        setDrafts((previous) => [...previous, emptyRoleDraft()]);
    };
    return { change, drafts, handleAdd, remove };
}

function useCreate(onCreated: () => void) {
    const [creating, setCreating] = useState(false);
    const [error, setError] = useState('');
    const create = async (name: string, description: string, drafts: RoleDraft[]) => {
        const organization = organizationOf(name, description, drafts);
        if (typeof organization === 'string') {
            setError(organization);
            return;
        }
        setCreating(true);
        setError('');
        const result = await apiAction('/api/organizations', organization);
        setCreating(false);
        if (result.ok) {
            onCreated();
        } else {
            setError(result.error ?? 'Failed to create organization');
        }
    };
    return { create, creating, error };
}

function Field({
    label,
    value,
    placeholder,
    first = false,
    onValue,
}: {
    label: string;
    value: string;
    placeholder: string;
    first?: boolean;
    onValue: (value: string) => void;
}) {
    const id = useId();
    return (
        <div>
            <label className={LABEL} htmlFor={id}>
                {label}
            </label>
            <input
                className={FIELD}
                id={id}
                onChange={(event) => {
                    onValue(event.target.value);
                }}
                placeholder={placeholder}
                ref={first ? focusOnMount : undefined}
                value={value}
            />
        </div>
    );
}

function Roles({ roles }: { roles: ReturnType<typeof useRoleDrafts> }) {
    const names = roles.drafts.map((draft) => draft.name.trim()).filter((name) => name !== '');
    return (
        <div>
            <p className={LABEL}>Roles</p>
            <div className="space-y-3">
                {roles.drafts.map((draft, index) => (
                    <RoleDraftCard
                        draft={draft}
                        index={index}
                        key={draft.id}
                        onChange={(patch) => {
                            roles.change(draft.id, patch);
                        }}
                        onRemove={() => {
                            roles.remove(draft.id);
                        }}
                        removable={roles.drafts.length > 1}
                        superiors={names.filter((name) => name !== draft.name.trim())}
                    />
                ))}
            </div>
            <button
                className="text-muted-foreground/50 hover:text-foreground/70 mt-2 flex items-center gap-1.5 font-mono text-[10px] transition-colors"
                onClick={roles.handleAdd}
                type="button"
            >
                <IconPlus size={12} /> Add Role
            </button>
        </div>
    );
}

/** Declares an organization: its name, what it is for, and its roles. */
export function CreateOrganizationDialog({
    onClose,
    onCreated,
}: {
    onClose: () => void;
    onCreated: () => void;
}) {
    const [name, setName] = useState('');
    const [description, setDescription] = useState('');
    const roles = useRoleDrafts();
    const { create, creating, error } = useCreate(onCreated);

    return (
        <Modal
            className="flex max-h-[85vh] max-w-lg flex-col"
            closable={!creating}
            onClose={onClose}
        >
            <div className="shrink-0 px-6 pt-6 pb-4">
                <h2 className="font-heading text-foreground/90 text-lg">New Organization</h2>
                <p className="text-muted-foreground/40 mt-0.5 text-[11px]">
                    Define a role structure for organizing agents
                </p>
            </div>
            <div className="min-h-0 space-y-4 overflow-y-auto px-6 pb-6">
                <Field
                    first
                    label="Name"
                    onValue={setName}
                    placeholder="Military Chain"
                    value={name}
                />
                <Field
                    label="Description"
                    onValue={setDescription}
                    placeholder="A strict top-down command structure"
                    value={description}
                />
                <Roles roles={roles} />
                {error !== '' && <p className="text-xs text-red-400/80">{error}</p>}
                <ModalActions
                    busy={creating}
                    busyLabel="Creating…"
                    label="Create"
                    onCancel={onClose}
                    onConfirm={() => {
                        void create(name, description, roles.drafts);
                    }}
                />
            </div>
        </Modal>
    );
}
