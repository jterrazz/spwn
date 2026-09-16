'use client';

import { IconCheck, IconPencil, IconX } from '@tabler/icons-react';
import { useEffect, useRef, useState } from 'react';

interface InlineEditProps {
    value: string;
    placeholder?: string;
    onSave: (value: string) => Promise<boolean>;
    multiline?: boolean;
    className?: string;
    editClassName?: string;
}

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
    const inputRef = useRef<HTMLInputElement | HTMLTextAreaElement>(null);

    useEffect(() => {
        setDraft(value);
    }, [value]);

    useEffect(() => {
        if (editing && inputRef.current) {
            inputRef.current.focus();
            inputRef.current.select();
        }
    }, [editing]);

    const handleSave = async () => {
        if (draft === value) {
            setEditing(false);
            return;
        }
        setSaving(true);
        const ok = await onSave(draft);
        setSaving(false);
        if (ok) {
            setEditing(false);
        }
    };

    const handleCancel = () => {
        setDraft(value);
        setEditing(false);
    };

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter' && !multiline) {
            e.preventDefault();
            handleSave();
        }
        if (e.key === 'Enter' && multiline && (e.metaKey || e.ctrlKey)) {
            e.preventDefault();
            handleSave();
        }
        if (e.key === 'Escape') {
            handleCancel();
        }
    };

    if (editing) {
        return (
            <div className={`flex items-start gap-2 ${editClassName}`}>
                {multiline ? (
                    <textarea
                        className="text-foreground/80 flex-1 resize-none rounded-lg border border-white/[0.12] bg-white/[0.03] px-3 py-2 text-sm transition-colors focus:border-white/[0.2] focus:outline-none"
                        onChange={(e) => setDraft(e.target.value)}
                        onKeyDown={handleKeyDown}
                        placeholder={placeholder}
                        ref={inputRef as React.RefObject<HTMLTextAreaElement>}
                        rows={3}
                        value={draft}
                    />
                ) : (
                    <input
                        className="text-foreground/80 flex-1 rounded-lg border border-white/[0.12] bg-white/[0.03] px-3 py-2 text-sm transition-colors focus:border-white/[0.2] focus:outline-none"
                        onChange={(e) => setDraft(e.target.value)}
                        onKeyDown={handleKeyDown}
                        placeholder={placeholder}
                        ref={inputRef as React.RefObject<HTMLInputElement>}
                        value={draft}
                    />
                )}
                <button
                    className="rounded-lg p-1.5 text-green-400/70 transition-colors hover:bg-green-500/10 hover:text-green-400 disabled:opacity-30"
                    disabled={saving}
                    onClick={handleSave}
                >
                    {saving ? (
                        <div className="border-foreground/30 border-t-foreground/70 h-3.5 w-3.5 animate-spin rounded-full border-2" />
                    ) : (
                        <IconCheck size={14} />
                    )}
                </button>
                <button
                    className="text-muted-foreground/40 hover:text-foreground/60 rounded-lg p-1.5 transition-colors hover:bg-white/[0.04]"
                    onClick={handleCancel}
                >
                    <IconX size={14} />
                </button>
            </div>
        );
    }

    return (
        <div
            className={`group relative cursor-pointer ${className}`}
            onClick={() => setEditing(true)}
        >
            <span className={value ? '' : 'text-muted-foreground/30 italic'}>
                {value || placeholder}
            </span>
            <IconPencil
                className="ml-2 inline-block opacity-0 transition-opacity group-hover:opacity-40"
                size={12}
            />
        </div>
    );
}

interface InlineTagsEditProps {
    tags: string[];
    onSave: (tags: string[]) => Promise<boolean>;
    color?: string;
}

export function InlineTagsEdit({
    tags,
    onSave,
    color = 'bg-purple-500/10 text-purple-300/80 border-purple-500/20',
}: InlineTagsEditProps) {
    const [editing, setEditing] = useState(false);
    const [newTag, setNewTag] = useState('');
    const [saving, setSaving] = useState(false);
    const inputRef = useRef<HTMLInputElement>(null);

    useEffect(() => {
        if (editing && inputRef.current) {
            inputRef.current.focus();
        }
    }, [editing]);

    const handleAdd = async () => {
        const tag = newTag.trim();
        if (!tag || tags.includes(tag)) {
            setNewTag('');
            return;
        }
        setSaving(true);
        const ok = await onSave([...tags, tag]);
        setSaving(false);
        if (ok) {
            setNewTag('');
        }
    };

    const handleRemove = async (tag: string) => {
        setSaving(true);
        await onSave(tags.filter((t) => t !== tag));
        setSaving(false);
    };

    return (
        <div className="flex flex-wrap items-center gap-2">
            {tags.map((tag) => (
                <span
                    className={`rounded-full border px-2.5 py-1 font-mono text-[11px] ${color} ${editing ? 'pr-1.5' : ''}`}
                    key={tag}
                >
                    {tag}
                    {editing && (
                        <button
                            className="ml-1.5 text-red-400/60 transition-colors hover:text-red-400"
                            disabled={saving}
                            onClick={async () => await handleRemove(tag)}
                        >
                            <IconX size={10} />
                        </button>
                    )}
                </span>
            ))}
            {editing ? (
                <div className="flex items-center gap-1">
                    <input
                        className="text-foreground/70 w-24 rounded-full border border-white/[0.12] bg-white/[0.03] px-2.5 py-1 font-mono text-[11px] transition-colors focus:border-white/[0.2] focus:outline-none"
                        onChange={(e) => setNewTag(e.target.value)}
                        onKeyDown={(e) => {
                            if (e.key === 'Enter') {
                                e.preventDefault();
                                handleAdd();
                            }
                            if (e.key === 'Escape') {
                                setEditing(false);
                            }
                        }}
                        placeholder="new trait..."
                        ref={inputRef}
                        value={newTag}
                    />
                    <button
                        className="text-muted-foreground/40 hover:text-foreground/60 p-1 transition-colors"
                        onClick={() => setEditing(false)}
                    >
                        <IconCheck size={12} />
                    </button>
                </div>
            ) : (
                <button
                    className="text-muted-foreground/30 hover:text-muted-foreground/50 rounded-full border border-dashed border-white/[0.1] px-2.5 py-1 font-mono text-[11px] transition-colors hover:border-white/[0.2]"
                    onClick={() => setEditing(true)}
                >
                    + add
                </button>
            )}
        </div>
    );
}
