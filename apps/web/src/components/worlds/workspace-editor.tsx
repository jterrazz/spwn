'use client';

import { LABEL, workspaceDraft, WorkspaceRow } from './spawn-world-parts';
import type { WorkspaceDraft } from './spawn-world-parts';

interface WorkspaceEditorProps {
    workspaces: WorkspaceDraft[];
    defaultPath: string;
    onChange: (workspaces: WorkspaceDraft[]) => void;
}

/** The host folders a new world mounts — none makes it ephemeral — each named, pathed and read-only or not. */
export function WorkspaceEditor({ workspaces, defaultPath, onChange }: WorkspaceEditorProps) {
    const add = () => {
        onChange([
            ...workspaces,
            workspaceDraft(workspaces.length === 0 ? 'default' : `w${workspaces.length}`),
        ]);
    };

    return (
        <div>
            <div className="mb-1.5 flex items-baseline justify-between">
                <span className={LABEL}>
                    Workspaces{' '}
                    {workspaces.length === 0 && (
                        <span className="text-muted-foreground/25 tracking-normal normal-case">
                            (ephemeral)
                        </span>
                    )}
                </span>
                <button
                    className="text-muted-foreground/50 hover:text-foreground/80 text-[10px] transition-colors"
                    onClick={add}
                    type="button"
                >
                    + Add
                </button>
            </div>
            {workspaces.length === 0 ? (
                <button
                    className="text-muted-foreground/40 hover:text-foreground/60 w-full rounded-lg border border-dashed border-white/[0.08] bg-white/[0.02] px-3 py-2.5 text-left text-[11px] transition-colors hover:border-white/[0.15]"
                    onClick={add}
                    type="button"
                >
                    Ephemeral world - click to add a host mount
                </button>
            ) : (
                <div className="space-y-2">
                    {workspaces.map((draft, index) => (
                        <WorkspaceRow
                            draft={draft}
                            key={draft.id}
                            onChange={(changed) => {
                                onChange(
                                    workspaces.map((entry) =>
                                        entry.id === changed.id ? changed : entry,
                                    ),
                                );
                            }}
                            onRemove={() => {
                                onChange(workspaces.filter((entry) => entry.id !== draft.id));
                            }}
                            placeholder={index === 0 ? defaultPath : '/path/to/dir'}
                        />
                    ))}
                </div>
            )}
        </div>
    );
}
