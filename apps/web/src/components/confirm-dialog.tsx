'use client';

import { Modal } from './modal';

interface ConfirmDialogProps {
    title: string;
    children: React.ReactNode;
    confirm: string;
    confirming: string;
    busy: boolean;
    onConfirm: () => void;
    onCancel: () => void;
}

/** A destructive action asked once more: what it does, Cancel, and the red button that does it. */
export function ConfirmDialog({
    title,
    children,
    confirm,
    confirming,
    busy,
    onConfirm,
    onCancel,
}: ConfirmDialogProps) {
    return (
        <Modal className="max-w-sm p-6" onClose={onCancel}>
            <h3 className="font-heading text-foreground/90 mb-2 text-lg">{title}</h3>
            <p className="text-muted-foreground/50 mb-6 text-sm">{children}</p>
            <div className="flex justify-end gap-3">
                <button
                    className="text-muted-foreground/60 hover:text-foreground/80 rounded-lg px-4 py-2 text-sm transition-colors hover:bg-white/[0.04]"
                    onClick={onCancel}
                    type="button"
                >
                    Cancel
                </button>
                <button
                    className="rounded-lg border border-red-500/20 bg-red-500/20 px-4 py-2 text-sm text-red-400 transition-colors hover:bg-red-500/30 disabled:opacity-50"
                    disabled={busy}
                    onClick={onConfirm}
                    type="button"
                >
                    {busy ? (
                        <span className="flex items-center gap-2">
                            <span className="h-3 w-3 animate-spin rounded-full border-2 border-red-400/40 border-t-red-400" />
                            {confirming}
                        </span>
                    ) : (
                        confirm
                    )}
                </button>
            </div>
        </Modal>
    );
}
