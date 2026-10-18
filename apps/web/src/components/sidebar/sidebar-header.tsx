'use client';

import { IconSearch } from '@tabler/icons-react';
import Link from 'next/link';

import { SidebarHeader } from '../ui/sidebar';
import { useConnection } from './use-connection';

function openPalette(): void {
    globalThis.dispatchEvent(
        new KeyboardEvent('keydown', { bubbles: true, key: 'k', metaKey: true }),
    );
}

/** The logo (its hexagon green while the API answers) and the button that opens the command palette. */
export function SpwnSidebarHeader() {
    const connection = useConnection();
    const connected = connection === 'connected';
    return (
        <SidebarHeader className="gap-0 pt-4 pb-4">
            <div className="flex items-center justify-between gap-2 px-2">
                <div className="group/logo flex min-w-0 flex-1 items-center">
                    <Link className="flex items-center gap-1.5" href="/">
                        <span
                            className={`font-heading text-base transition-colors ${connected ? 'text-green-500' : 'text-red-400'}`}
                        >
                            ⬡
                        </span>
                        <span className="font-heading text-foreground text-base tracking-[0.12em]">
                            spwn
                        </span>
                    </Link>
                    <span
                        className={`ml-auto font-mono text-[10px] tracking-wider uppercase opacity-0 transition-opacity group-hover/logo:opacity-100 ${connected ? 'text-green-500/60' : 'text-red-400/60'}`}
                    >
                        {connection}
                    </span>
                </div>
                <button
                    aria-label="Search (⌘K)"
                    className="text-muted-foreground/30 hover:text-foreground flex h-8 w-8 shrink-0 items-center justify-center rounded-full transition-colors"
                    onClick={openPalette}
                    type="button"
                >
                    <IconSearch size={16} stroke={2.5} />
                </button>
            </div>
        </SidebarHeader>
    );
}
