'use client';

import { IconCheck, IconPencil, IconX } from '@tabler/icons-react';
import { useState } from 'react';

import { inBackground } from '../events/in-background';

const FIELD =
    'text-foreground/80 flex-1 rounded-lg border border-white/[0.12] bg-white/[0.03] px-3 py-2 text-sm transition-colors focus:border-white/[0.2] focus:outline-none';

interface InlineEditProps {
    value: string;
    placeholder?: string;
    onSave: (value: string) => Promise<boolean>;
    multiline?: boolean;
    className?: string;
    editClassName?: string;
}

function selectOnMount(element: HTMLInputElement | HTMLTextAreaElement | null): void {
    element?.focus();
    element?.select();
}

interface DraftFieldProps {
    draft: string;
    multiline: boolean;
    placeholder: string;
    onDraft: (draft: string) => void;
    onSave: () => void;
    onCancel: () => void;
}

function DraftField({ draft, multiline, placeholder, onDraft, onSave, onCancel }: DraftFieldProps) {
    const onKeyDown = (event: React.KeyboardEvent) => {
        if (event.key === 'Enter' && (!multiline || event.metaKey || event.ctrlKey)) {
            event.preventDefault();
            onSave();
        } else if (event.key === 'Escape') {
            onCancel();
        }
    };
    const shared = { className: FIELD, onKeyDown, placeholder, value: draft };
    return multiline ? (
        <textarea
            {...shared}
            className={`${FIELD} resize-none`}
            onChange={(event) => {
                onDraft(event.target.value);
            }}
            ref={selectOnMount}
            rows={3}
        />
    ) : (
        <input
            {...shared}
            onChange={(event) => {
                onDraft(event.target.value);
            }}
            ref={selectOnMount}
        />
    );
}

function EditActions({
    saving,
    onSave,
    onCancel,
}: {
    saving: boolean;
    onSave: () => void;
    onCancel: () => void;
}) {
    return (
        <>
            <button
                aria-label="Save"
                className="rounded-lg p-1.5 text-green-400/70 transition-colors hover:bg-green-500/10 hover:text-green-400 disabled:opacity-30"
                disabled={saving}
                onClick={onSave}
                type="button"
            >
                {saving ? (
                    <div className="border-foreground/30 border-t-foreground/70 h-3.5 w-3.5 animate-spin rounded-full border-2" />
                ) : (
                    <IconCheck size={14} />
                )}
            </button>
            <button
                aria-label="Cancel"
                className="text-muted-foreground/40 hover:text-foreground/60 rounded-lg p-1.5 transition-colors hover:bg-white/[0.04]"
                onClick={onCancel}
                type="button"
            >
                <IconX size={14} />
            </button>
        </>
    );
}

function ShownValue({
    value,
    placeholder,
    className,
    onEdit,
}: {
    value: string;
    placeholder: string;
    className: string;
    onEdit: () => void;
}) {
    return (
        <button
            className={`group relative block w-full cursor-pointer text-left ${className}`}
            onClick={onEdit}
            type="button"
        >
            <span className={value === '' ? 'text-muted-foreground/30 italic' : ''}>
                {value === '' ? placeholder : value}
            </span>
            <IconPencil
                className="ml-2 inline-block opacity-0 transition-opacity group-hover:opacity-40"
                size={12}
            />
        </button>
    );
}

/** A value shown as text that turns into a field on click: Enter (⌘Enter when multiline) saves, Escape cancels. */
export function InlineEdit({
    value,
    placeholder = 'Click to edit...',
    onSave,
    multiline = false,
    className = '',
    editClassName = '',
}: InlineEditProps) {
    const [editing, setEditing] = useState(false);
    const [draft, setDraft] = useState(value);
    const [saving, setSaving] = useState(false);

    const save = async () => {
        if (draft !== value) {
            setSaving(true);
            const saved = await onSave(draft);
            setSaving(false);
            if (!saved) {
                return;
            }
        }
        setEditing(false);
    };
    const cancel = () => {
        setEditing(false);
    };

    if (!editing) {
        return (
            <ShownValue
                className={className}
                onEdit={() => {
                    setDraft(value);
                    setEditing(true);
                }}
                placeholder={placeholder}
                value={value}
            />
        );
    }
    return (
        <div className={`flex items-start gap-2 ${editClassName}`}>
            <DraftField
                draft={draft}
                multiline={multiline}
                onCancel={cancel}
                onDraft={setDraft}
                onSave={inBackground(save)}
                placeholder={placeholder}
            />
            <EditActions onCancel={cancel} onSave={inBackground(save)} saving={saving} />
        </div>
    );
}
