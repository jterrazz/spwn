'use client';

import { IconX } from '@tabler/icons-react';
import { useId } from 'react';

import type { RoleDraft } from './role-draft';

const INPUT =
    'text-foreground/80 placeholder:text-muted-foreground/25 w-full rounded border border-white/[0.08] bg-white/[0.03] px-2 py-1.5 font-mono text-xs transition-colors focus:border-white/[0.15] focus:outline-none';
const CAPTION = 'text-muted-foreground/25 mb-0.5 block font-mono text-[8px] uppercase';

interface RoleDraftCardProps {
    draft: RoleDraft;
    index: number;
    // value: the other roles' names, the ones this role may report to
    superiors: string[];
    removable: boolean;
    onChange: (patch: Partial<RoleDraft>) => void;
    onRemove: () => void;
}

function TextField({
    label,
    value,
    placeholder,
    onValue,
}: {
    label: string;
    value: string;
    placeholder: string;
    onValue: (value: string) => void;
}) {
    const id = useId();
    return (
        <div>
            <label className={CAPTION} htmlFor={id}>
                {label}
            </label>
            <input
                className={INPUT}
                id={id}
                onChange={(event) => {
                    onValue(event.target.value);
                }}
                placeholder={placeholder}
                value={value}
            />
        </div>
    );
}

function NameAndLevel({
    draft,
    index,
    onChange,
}: Pick<RoleDraftCardProps, 'draft' | 'index' | 'onChange'>) {
    return (
        <div className="grid grid-cols-[1fr_60px] gap-2">
            <input
                aria-label={`Role ${index + 1} name`}
                className={INPUT}
                onChange={(event) => {
                    onChange({ name: event.target.value });
                }}
                placeholder="Role name"
                value={draft.name}
            />
            <input
                aria-label={`Role ${index + 1} level`}
                className={`${INPUT} text-center`}
                onChange={(event) => {
                    onChange({ level: Number.parseInt(event.target.value, 10) || 0 });
                }}
                placeholder="Lvl"
                type="number"
                value={draft.level}
            />
        </div>
    );
}

function ReportsTo({
    draft,
    superiors,
    onChange,
}: Pick<RoleDraftCardProps, 'draft' | 'onChange' | 'superiors'>) {
    const id = useId();
    return (
        <div>
            <label className={CAPTION} htmlFor={id}>
                Reports to
            </label>
            <select
                className={INPUT}
                id={id}
                onChange={(event) => {
                    onChange({ reportsTo: event.target.value });
                }}
                value={draft.reportsTo}
            >
                <option value="">None</option>
                {superiors.map((name) => (
                    <option key={name} value={name}>
                        {name}
                    </option>
                ))}
            </select>
        </div>
    );
}

/** One role of the creation form: name, level, superior, the roles it commands and its permissions. */
export function RoleDraftCard({
    draft,
    index,
    superiors,
    removable,
    onChange,
    onRemove,
}: RoleDraftCardProps) {
    return (
        <div className="space-y-2 rounded-lg border border-white/[0.06] bg-white/[0.01] p-3">
            <div className="flex items-center justify-between gap-2">
                <span className="text-muted-foreground/30 font-mono text-[9px]">#{index + 1}</span>
                {removable && (
                    <button
                        aria-label={`Remove role ${index + 1}`}
                        className="text-muted-foreground/30 transition-colors hover:text-red-400/70"
                        onClick={onRemove}
                        type="button"
                    >
                        <IconX size={14} />
                    </button>
                )}
            </div>
            <NameAndLevel draft={draft} index={index} onChange={onChange} />
            <div className="grid grid-cols-2 gap-2">
                <ReportsTo draft={draft} onChange={onChange} superiors={superiors} />
                <TextField
                    label="Can command"
                    onValue={(canCommand) => {
                        onChange({ canCommand });
                    }}
                    placeholder="role1, role2"
                    value={draft.canCommand}
                />
            </div>
            <TextField
                label="Permissions"
                onValue={(permissions) => {
                    onChange({ permissions });
                }}
                placeholder="delegate, review, execute"
                value={draft.permissions}
            />
        </div>
    );
}
