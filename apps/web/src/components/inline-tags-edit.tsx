'use client';

import { IconCheck, IconX } from '@tabler/icons-react';
import { useState } from 'react';

import { inBackground } from '../events/in-background';
import { focusOnMount } from './focus-on-mount';

interface InlineTagsEditProps {
    tags: string[];
    onSave: (tags: string[]) => Promise<boolean>;
    color?: string;
}

function TagInput({
    onAdd,
    onDone,
}: {
    onAdd: (tag: string) => Promise<boolean>;
    onDone: () => void;
}) {
    const [tag, setTag] = useState('');
    const add = async () => {
        if (await onAdd(tag.trim())) {
            setTag('');
        }
    };
    return (
        <div className="flex items-center gap-1">
            <input
                aria-label="New trait"
                className="text-foreground/70 w-24 rounded-full border border-white/[0.12] bg-white/[0.03] px-2.5 py-1 font-mono text-[11px] transition-colors focus:border-white/[0.2] focus:outline-none"
                onChange={(event) => {
                    setTag(event.target.value);
                }}
                onKeyDown={(event) => {
                    if (event.key === 'Enter') {
                        event.preventDefault();
                        void add();
                    } else if (event.key === 'Escape') {
                        onDone();
                    }
                }}
                placeholder="new trait..."
                ref={focusOnMount}
                value={tag}
            />
            <button
                aria-label="Done"
                className="text-muted-foreground/40 hover:text-foreground/60 p-1 transition-colors"
                onClick={onDone}
                type="button"
            >
                <IconCheck size={12} />
            </button>
        </div>
    );
}

function TagChip({
    tag,
    color,
    editing,
    saving,
    onRemove,
}: {
    tag: string;
    color: string;
    editing: boolean;
    saving: boolean;
    onRemove: () => void;
}) {
    return (
        <span
            className={`rounded-full border px-2.5 py-1 font-mono text-[11px] ${color} ${editing ? 'pr-1.5' : ''}`}
        >
            {tag}
            {editing && (
                <button
                    aria-label={`Remove ${tag}`}
                    className="ml-1.5 text-red-400/60 transition-colors hover:text-red-400"
                    disabled={saving}
                    onClick={onRemove}
                    type="button"
                >
                    <IconX size={10} />
                </button>
            )}
        </span>
    );
}

/** A list of tags edited in place: each removable, a new one added on Enter. */
export function InlineTagsEdit({
    tags,
    onSave,
    color = 'bg-purple-500/10 text-purple-300/80 border-purple-500/20',
}: InlineTagsEditProps) {
    const [editing, setEditing] = useState(false);
    const [saving, setSaving] = useState(false);

    const save = async (next: string[]) => {
        setSaving(true);
        const saved = await onSave(next);
        setSaving(false);
        return saved;
    };
    const add = async (tag: string) =>
        tag === '' || tags.includes(tag) ? true : await save([...tags, tag]);

    return (
        <div className="flex flex-wrap items-center gap-2">
            {tags.map((tag) => (
                <TagChip
                    color={color}
                    editing={editing}
                    key={tag}
                    onRemove={inBackground(async () => {
                        await save(tags.filter((kept) => kept !== tag));
                    })}
                    saving={saving}
                    tag={tag}
                />
            ))}
            {editing ? (
                <TagInput
                    onAdd={add}
                    onDone={() => {
                        setEditing(false);
                    }}
                />
            ) : (
                <button
                    className="text-muted-foreground/30 hover:text-muted-foreground/50 rounded-full border border-dashed border-white/[0.1] px-2.5 py-1 font-mono text-[11px] transition-colors hover:border-white/[0.2]"
                    onClick={() => {
                        setEditing(true);
                    }}
                    type="button"
                >
                    + add
                </button>
            )}
        </div>
    );
}
