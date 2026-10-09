'use client';

import { IconAlertTriangle } from '@tabler/icons-react';

import { useProgressMessages } from '../../hooks/use-progress-messages';
import { ProgressShimmer } from '../progress-shimmer';

const DESTROY_PROGRESS = [
    { after: 0, text: 'Destroying world...' },
    { after: 5, text: 'Stopping containers...' },
    { after: 15, text: 'Cleaning up...' },
];

interface DestroyConfirmProps {
    worldId: string;
    destroying: boolean;
    onDestroy: () => void;
    onCancel: () => void;
}

/** The inline warning before a world is destroyed, with the progress of its teardown. */
export function DestroyConfirm({ worldId, destroying, onDestroy, onCancel }: DestroyConfirmProps) {
    const progress = useProgressMessages(destroying, DESTROY_PROGRESS);
    return (
        <div className="rounded-xl border border-red-500/30 bg-red-500/5 p-5">
            <div className="flex items-start gap-3">
                <IconAlertTriangle className="mt-0.5 shrink-0 text-red-400" size={20} />
                <div className="flex-1">
                    <h3 className="font-heading text-sm text-red-300">Destroy World?</h3>
                    <p className="mt-1 text-xs text-red-300/60">
                        This will permanently destroy <span className="font-mono">{worldId}</span>{' '}
                        and all its agents. This action cannot be undone.
                    </p>
                    {destroying && (
                        <ProgressShimmer active className="mt-2 mb-1" message={progress} />
                    )}
                    <div className="mt-4 flex gap-2">
                        <button
                            className="rounded-lg border border-red-500/30 bg-red-500/20 px-4 py-2 text-xs text-red-300 transition-colors hover:bg-red-500/30 disabled:opacity-30"
                            disabled={destroying}
                            onClick={onDestroy}
                            type="button"
                        >
                            {destroying ? (
                                <span className="flex items-center gap-2">
                                    <span className="h-3 w-3 animate-spin rounded-full border-2 border-red-300/40 border-t-red-300" />
                                    {progress}
                                </span>
                            ) : (
                                'Yes, destroy it'
                            )}
                        </button>
                        <button
                            className="text-muted-foreground/50 hover:text-foreground/70 rounded-lg px-4 py-2 text-xs transition-colors hover:bg-white/[0.04]"
                            onClick={onCancel}
                            type="button"
                        >
                            Cancel
                        </button>
                    </div>
                </div>
            </div>
        </div>
    );
}
