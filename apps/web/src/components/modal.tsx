'use client';

interface ModalProps {
    onClose: () => void;
    // invariant: false while the modal's work runs, so the backdrop cannot drop it
    closable?: boolean;
    className?: string;
    children: React.ReactNode;
}

/** A centred panel over a dimmed page; the backdrop is the button that closes it. */
export function Modal({ onClose, closable = true, className = 'max-w-md', children }: ModalProps) {
    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
            <button
                aria-label="Close"
                className="absolute inset-0 bg-black/40 backdrop-blur-sm"
                disabled={!closable}
                onClick={onClose}
                type="button"
            />
            <div
                className={`bg-popover/95 relative z-10 mx-4 w-full overflow-hidden rounded-2xl border border-white/[0.08] shadow-2xl backdrop-blur-md ${className}`}
            >
                {children}
            </div>
        </div>
    );
}

/** The thin bar sliding across a modal's top edge while its work runs. */
export function WorkingStripe({ active }: { active: boolean }) {
    if (!active) {
        return null;
    }
    return (
        <div className="h-0.5 w-full overflow-hidden bg-white/[0.04]">
            <div
                className="h-full w-1/3 rounded-full bg-emerald-500/30"
                style={{ animation: 'progressSlide 1.5s ease-in-out infinite' }}
            />
        </div>
    );
}

interface ModalActionsProps {
    busy: boolean;
    label: string;
    busyLabel: string;
    onCancel: () => void;
    onConfirm: () => void;
}

/** A modal's closing row: Cancel, and the button that does the modal's work, spinning while it runs. */
export function ModalActions({ busy, label, busyLabel, onCancel, onConfirm }: ModalActionsProps) {
    return (
        <div className="mt-6 flex justify-end gap-3">
            <button
                className="text-muted-foreground/60 hover:text-foreground/80 rounded-lg px-4 py-2 text-sm transition-colors hover:bg-white/[0.04] disabled:opacity-50"
                disabled={busy}
                onClick={onCancel}
                type="button"
            >
                Cancel
            </button>
            <button
                className="text-foreground/90 flex items-center gap-2 rounded-lg border border-white/[0.08] bg-white/[0.1] px-4 py-2 text-sm transition-colors hover:bg-white/[0.16] disabled:opacity-50"
                disabled={busy}
                onClick={onConfirm}
                type="button"
            >
                {busy && (
                    <div className="border-foreground/30 border-t-foreground/80 h-3 w-3 animate-spin rounded-full border-2" />
                )}
                {busy ? busyLabel : label}
            </button>
        </div>
    );
}
