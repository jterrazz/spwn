'use client';

import { IconX } from '@tabler/icons-react';

import { useWorldLogs } from './use-world';
import type { LogLine } from './use-world';

export type Panel = 'logs' | 'snapshots';

function clockOf(timestamp: string): string {
    return new Date(timestamp).toLocaleTimeString([], {
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
    });
}

function Logs({
    worldId,
    lines,
    loading,
}: {
    worldId: string;
    lines: LogLine[];
    loading: boolean;
}) {
    return (
        <div className="space-y-0.5 p-4 font-mono text-[11px]">
            {loading && lines.length === 0 && (
                <div className="text-muted-foreground/30 flex items-center justify-center gap-2 py-8">
                    <div className="border-foreground/20 border-t-foreground/50 h-3 w-3 animate-spin rounded-full border-2" />
                    <span className="text-sm">Connecting to log stream...</span>
                </div>
            )}
            {!loading && lines.length === 0 && (
                <div className="py-8 text-center">
                    <p className="text-muted-foreground/30 text-sm">No logs available</p>
                    <p className="text-muted-foreground/20 mt-1 font-mono text-[10px]">
                        Use the CLI: spwn logs {worldId}
                    </p>
                </div>
            )}
            {lines.map((line) => (
                <div
                    className="border-border/10 flex gap-2 border-b py-1.5 last:border-0"
                    key={`${line.timestamp}-${line.source}-${line.message}`}
                >
                    <span className="text-muted-foreground/25 w-14 shrink-0">
                        {clockOf(line.timestamp)}
                    </span>
                    <span className="w-10 shrink-0 text-blue-400/70 uppercase">info</span>
                    <span className="text-muted-foreground/40 w-16 shrink-0">{line.source}</span>
                    <span className="text-foreground/60 break-all">{line.message}</span>
                </div>
            ))}
        </div>
    );
}

/** The side column of a world's page: its activity log, or its snapshots (which the CLI lists). */
export function WorldSidePanel({
    panel,
    worldId,
    onClose,
}: {
    panel: Panel;
    worldId: string;
    onClose: () => void;
}) {
    const { lines, loading } = useWorldLogs(worldId, panel === 'logs');
    return (
        <div className="border-border/30 hidden w-96 shrink-0 flex-col overflow-hidden border-l md:flex">
            <div className="border-border/30 flex shrink-0 items-center justify-between border-b px-5 py-4">
                <h2 className="font-heading text-foreground/80 text-sm capitalize">{panel}</h2>
                <button
                    aria-label="Close"
                    className="text-muted-foreground/40 hover:text-foreground/70 transition-colors"
                    onClick={onClose}
                    type="button"
                >
                    <IconX size={16} />
                </button>
            </div>
            <div className="flex-1 overflow-y-auto">
                {panel === 'logs' ? (
                    <Logs lines={lines} loading={loading} worldId={worldId} />
                ) : (
                    <div className="py-8 text-center">
                        <p className="text-muted-foreground/30 text-sm">No snapshots</p>
                        <p className="text-muted-foreground/20 mt-1 font-mono text-[10px]">
                            Use the CLI: spwn world snap ls {worldId}
                        </p>
                    </div>
                )}
            </div>
        </div>
    );
}
