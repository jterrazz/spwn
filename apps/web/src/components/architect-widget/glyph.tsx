'use client';

import { IconHexagonFilled } from '@tabler/icons-react';

export function ArchitectGlyph({ isRunning, isActive }: { isRunning: boolean; isActive: boolean }) {
    return (
        <span className="relative inline-flex h-[18px] w-[18px] shrink-0 items-center justify-center self-center overflow-visible">
            <style jsx>{`
                @keyframes architect-rainbow-hue {
                    0% {
                        filter: hue-rotate(0deg) brightness(1.2);
                    }
                    100% {
                        filter: hue-rotate(360deg) brightness(1.2);
                    }
                }
            `}</style>
            {isRunning && isActive ? (
                <>
                    <IconHexagonFilled
                        className="absolute inset-0 m-auto text-pink-400 opacity-70 blur-[6px]"
                        size={14}
                        style={{ animation: 'architect-rainbow-hue 3s linear infinite' }}
                    />
                    <IconHexagonFilled
                        className="absolute inset-0 m-auto text-pink-400"
                        size={14}
                        style={{ animation: 'architect-rainbow-hue 3s linear infinite' }}
                    />
                </>
            ) : (
                <span className="block translate-y-[0.5px] leading-none">
                    <IconHexagonFilled className="text-muted-foreground/45" size={14} />
                </span>
            )}
        </span>
    );
}
