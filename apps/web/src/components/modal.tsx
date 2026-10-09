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
