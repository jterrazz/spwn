'use client';

import { useId, useState } from 'react';

import { apiAction } from '../../api/client';
import type { World } from '../../domain/model';
import { inBackground } from '../../events/in-background';
import { focusOnMount } from '../focus-on-mount';
import { Modal, ModalActions } from '../modal';

interface RenameDialogProps {
    world: World;
    onRenamed: () => void;
    onFailed: (message: string) => void;
    onClose: () => void;
}

/** Renames a world; an empty name falls back to the one derived from its id. */
export function RenameDialog({ world, onRenamed, onFailed, onClose }: RenameDialogProps) {
    const id = useId();
    const [name, setName] = useState(world.name ?? '');
    const [renaming, setRenaming] = useState(false);

    const rename = async () => {
        setRenaming(true);
        const result = await apiAction(`/api/worlds/${world.id}`, { name: name.trim() }, 'PATCH');
        setRenaming(false);
        if (result.ok) {
            onRenamed();
        } else {
            onFailed(`Error: ${result.error ?? 'Failed to rename'}`);
        }
    };

    return (
        <Modal className="max-w-md p-6" closable={!renaming} onClose={onClose}>
            <h3 className="font-heading text-foreground/90 mb-1 text-lg">Rename World</h3>
            <label className="text-muted-foreground/50 mb-5 block text-sm" htmlFor={id}>
                Leave empty to fall back to the auto-generated name (
                <span className="font-mono">{world.id.split('-')[1] ?? world.id}</span>).
            </label>
            <input
                className="text-foreground/80 placeholder:text-muted-foreground/30 w-full rounded-lg border border-white/[0.08] bg-white/[0.04] px-3 py-2.5 text-sm transition-colors focus:border-white/[0.16] focus:outline-none disabled:opacity-50"
                disabled={renaming}
                id={id}
                onChange={(event) => {
                    setName(event.target.value);
                }}
                onKeyDown={(event) => {
                    if (event.key === 'Enter') {
                        void rename();
                    }
                }}
                placeholder="My Project"
                ref={focusOnMount}
                type="text"
                value={name}
            />
            <ModalActions
                busy={renaming}
                busyLabel="Saving…"
                label="Save"
                onCancel={onClose}
                onConfirm={inBackground(rename)}
            />
        </Modal>
    );
}
