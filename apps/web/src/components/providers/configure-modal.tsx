'use client';

import { IconTerminal2, IconX } from '@tabler/icons-react';

import { Modal } from '../modal';
import { metaOf } from './meta';

/** How to connect a provider: sign in with its CLI on the host, then reconnect here. */
export function ConfigureModal({ provider, onClose }: { provider: string; onClose: () => void }) {
    const meta = metaOf(provider);
    return (
        <Modal className="max-w-md p-6" onClose={onClose}>
            <div className="mb-5 flex items-center justify-between">
                <h3 className="font-heading text-foreground/80 text-sm tracking-wide">
                    Connect {meta.name}
                </h3>
                <button
                    aria-label="Close"
                    className="text-muted-foreground/30 hover:text-muted-foreground/60 p-1 transition-colors"
                    onClick={onClose}
                    type="button"
                >
                    <IconX size={16} />
                </button>
            </div>
            <div className="space-y-4">
                <div className="space-y-3 rounded-lg border border-white/[0.06] bg-white/[0.02] p-4">
                    <p className="text-muted-foreground/50 text-xs leading-relaxed">
                        Sign in via the runtime CLI on your host machine. Spwn detects credentials
                        from your system keychain and shares them with all containers automatically.
                    </p>
                    {meta.signIn !== '' && (
                        <div className="text-foreground/50 flex items-center gap-2 rounded-lg border border-white/[0.06] bg-white/[0.03] px-3 py-2 font-mono text-[11px]">
                            <IconTerminal2
                                className="text-muted-foreground/30 shrink-0"
                                size={13}
                            />
                            <span>{meta.signIn}</span>
                        </div>
                    )}
                    <p className="text-muted-foreground/25 text-[10px]">
                        After signing in, click Reconnect in the settings to pick up the new
                        credentials.
                    </p>
                </div>
                <button
                    className="text-muted-foreground/50 hover:text-foreground/70 w-full rounded-xl px-4 py-2.5 text-sm transition-colors hover:bg-white/[0.04]"
                    onClick={onClose}
                    type="button"
                >
                    Done
                </button>
            </div>
        </Modal>
    );
}
