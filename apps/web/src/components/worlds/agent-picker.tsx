'use client';

import { useState } from 'react';

import { apiAction } from '../../api/client';
import { inBackground } from '../../events/in-background';
import { LABEL } from './spawn-world-parts';

interface AgentPickerProps {
    available: string[];
    selected: Set<string>;
    onToggle: (name: string) => void;
    onClear: () => void;
    onCreated: (name: string) => void;
    onError: (error: string) => void;
}

function AgentRows({
    available,
    selected,
    onToggle,
}: Pick<AgentPickerProps, 'available' | 'onToggle' | 'selected'>) {
    if (available.length === 0) {
        return (
            <p className="text-muted-foreground/40 px-3 py-2.5 text-[11px]">
                No agents yet - create one below or skip to spawn an empty world.
            </p>
        );
    }
    return (
        <ul className="divide-y divide-white/[0.04]">
            {available.map((name) => (
                <li key={name}>
                    <label className="flex cursor-pointer items-center gap-2.5 px-3 py-2 transition-colors hover:bg-white/[0.03]">
                        <input
                            checked={selected.has(name)}
                            className="accent-foreground h-3.5 w-3.5 cursor-pointer rounded border-white/[0.15] bg-white/[0.04]"
                            onChange={() => {
                                onToggle(name);
                            }}
                            type="checkbox"
                        />
                        <span
                            className={`font-mono text-sm ${selected.has(name) ? 'text-foreground/90' : 'text-foreground/60'}`}
                        >
                            {name}
                        </span>
                    </label>
                </li>
            ))}
        </ul>
    );
}

function NewAgent({ onCreated, onError }: Pick<AgentPickerProps, 'onCreated' | 'onError'>) {
    const [name, setName] = useState('');
    const [creating, setCreating] = useState(false);

    const create = async () => {
        const trimmed = name.trim();
        if (trimmed === '') {
            return;
        }
        setCreating(true);
        onError('');
        const result = await apiAction('/api/agents', { name: trimmed });
        setCreating(false);
        if (result.ok) {
            onCreated(trimmed);
            setName('');
        } else {
            onError(result.error ?? 'Failed to create agent');
        }
    };

    return (
        <div className="flex gap-2 border-t border-white/[0.06] px-3 py-2">
            <input
                aria-label="New agent name"
                className="text-foreground/80 placeholder:text-muted-foreground/30 flex-1 bg-transparent text-xs focus:outline-none"
                onChange={(event) => {
                    setName(event.target.value);
                }}
                onKeyDown={(event) => {
                    if (event.key === 'Enter') {
                        void create();
                    }
                }}
                placeholder="New agent name…"
                value={name}
            />
            <button
                className="text-foreground/70 shrink-0 rounded border border-white/[0.08] bg-white/[0.06] px-2.5 py-1 text-[11px] transition-all hover:bg-white/[0.1] disabled:cursor-not-allowed disabled:opacity-30"
                disabled={name.trim() === '' || creating}
                onClick={inBackground(create)}
                type="button"
            >
                {creating ? '…' : '+ Add'}
            </button>
        </div>
    );
}

/** The agents a new world deploys: the project's, checked one by one, and one created on the spot. */
export function AgentPicker({
    available,
    selected,
    onToggle,
    onClear,
    onCreated,
    onError,
}: AgentPickerProps) {
    return (
        <div>
            <div className="mb-1.5 flex items-baseline justify-between">
                <span className={LABEL}>
                    Agents{' '}
                    <span className="text-muted-foreground/25 tracking-normal normal-case">
                        ({selected.size === 0 ? 'none - empty world' : `${selected.size} selected`})
                    </span>
                </span>
                {selected.size > 0 && (
                    <button
                        className="text-muted-foreground/40 hover:text-foreground/70 text-[10px] transition-colors"
                        onClick={onClear}
                        type="button"
                    >
                        Clear
                    </button>
                )}
            </div>
            <div className="max-h-44 overflow-y-auto rounded-lg border border-white/[0.08] bg-white/[0.02]">
                <AgentRows available={available} onToggle={onToggle} selected={selected} />
                <NewAgent onCreated={onCreated} onError={onError} />
            </div>
        </div>
    );
}
