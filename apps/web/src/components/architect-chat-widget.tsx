'use client';

import { usePathname, useRouter } from 'next/navigation';
import { useCallback, useEffect, useRef, useState } from 'react';

import { useArchitectChat } from '../contexts/architect-chat-context';
import { ArchitectGlyph } from './architect-widget/glyph';
import { ArchitectPanel, PILL } from './architect-widget/panel';
import { ArchitectPill } from './architect-widget/pill';

function useFoldOnOutsidePointer(
    expanded: boolean,
    panel: React.RefObject<HTMLDivElement | null>,
    fold: () => void,
): void {
    useEffect(() => {
        if (!expanded) {
            return;
        }
        const onPointerDown = (event: MouseEvent | TouchEvent) => {
            if (event.target instanceof Node && panel.current?.contains(event.target) !== true) {
                fold();
            }
        };
        document.addEventListener('mousedown', onPointerDown);
        document.addEventListener('touchstart', onPointerDown, { passive: true });
        return () => {
            document.removeEventListener('mousedown', onPointerDown);
            document.removeEventListener('touchstart', onPointerDown);
        };
    }, [expanded, panel, fold]);
}

function OfflineButton({ onOpen }: { onOpen: () => void }) {
    return (
        <div className="animate-in fade-in fixed right-4 bottom-4 z-[200] duration-200">
            <button
                className={`flex items-center gap-2.5 rounded-full border px-4 py-2.5 transition-colors hover:bg-white/[0.08] ${PILL}`}
                onClick={onOpen}
                title="Architect is offline - click to start"
                type="button"
            >
                <ArchitectGlyph isActive={false} isRunning={false} />
                <span className="text-muted-foreground/35 text-[12px]">Architect offline</span>
            </button>
        </div>
    );
}

/** The architect in the corner of every page but its own: offline, folded into a prompt, or unfolded into a chat. */
export function ArchitectChatWidget() {
    const pathname = usePathname();
    const router = useRouter();
    const [expanded, setExpanded] = useState(false);
    const panelRef = useRef<HTMLDivElement>(null);
    const { messages, chatInput, setChatInput, sending, sendMessage, isRunning } =
        useArchitectChat();
    const fold = useCallback(() => {
        setExpanded(false);
    }, []);
    useFoldOnOutsidePointer(expanded, panelRef, fold);

    if (pathname === '/architect' || pathname.startsWith('/architect/')) {
        return null;
    }
    const send = () => {
        void sendMessage();
    };
    if (expanded) {
        return (
            <ArchitectPanel
                input={chatInput}
                messages={messages}
                onInput={setChatInput}
                onMinimize={fold}
                onOpenPage={() => {
                    fold();
                    router.push('/architect');
                }}
                onSend={send}
                panelRef={panelRef}
                running={isRunning}
                sending={sending}
            />
        );
    }
    if (!isRunning) {
        return (
            <OfflineButton
                onOpen={() => {
                    router.push('/architect');
                }}
            />
        );
    }
    return (
        <ArchitectPill
            input={chatInput}
            onExpand={() => {
                setExpanded(true);
            }}
            onInput={setChatInput}
            onSend={send}
            sending={sending}
        />
    );
}
