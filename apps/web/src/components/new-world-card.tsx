'use client';

import type { CSSProperties } from 'react';

import type { World } from '../domain/model';
import { Planet as PlanetGlobe } from './planet';

const PLACEHOLDER_WORLD: World = {
    id: 'w-new-00000',
    config: 'default',
    agent: '',
    agents: [],
    status: 'stopped',
    created_at: '',
};

interface NewWorldCardProps {
    onClick: () => void;
    opacity?: number;
    scale?: number;
    style?: CSSProperties;
    tint?: 'creating' | 'stopped';
}

/** Shared "create a new world" card - used both inline at the end of the carousel AND in the empty state. Wraps the placeholder PlanetGlobe in the exact same container, animations, and typography as every other world card, so the create affordance feels native to the list. */
export function NewWorldCard({
    onClick,
    opacity = 0.5,
    scale = 1,
    style,
    tint = 'stopped',
}: NewWorldCardProps) {
    const world: World =
        tint === 'creating' ? { ...PLACEHOLDER_WORLD, status: 'creating' } : PLACEHOLDER_WORLD;

    return (
        <div
            className="group/new-world relative flex shrink-0 cursor-pointer flex-col items-center"
            onClick={onClick}
            onMouseEnter={(e) => {
                e.currentTarget.style.opacity = '1';
                e.currentTarget.style.transform = `scale(${scale * 1.08})`;
            }}
            onMouseLeave={(e) => {
                e.currentTarget.style.opacity = String(opacity);
                e.currentTarget.style.transform = `scale(${scale})`;
            }}
            style={{
                opacity,
                transform: `scale(${scale})`,
                transition:
                    'opacity 320ms cubic-bezier(0.32, 0.72, 0.24, 1), ' +
                    'transform 420ms cubic-bezier(0.32, 0.72, 0.24, 1)',
                ...style,
            }}
        >
            <span
                aria-hidden
                className="pointer-events-none absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 rounded-full opacity-0 blur-2xl transition-opacity duration-500 ease-out group-hover/new-world:opacity-100"
                style={{
                    width: 180,
                    height: 180,
                    background:
                        'radial-gradient(circle, rgba(255,255,255,0.18) 0%, rgba(255,255,255,0.04) 45%, rgba(255,255,255,0) 75%)',
                }}
            />
            <PlanetGlobe
                compact
                hideLabels
                index={0}
                isSelected={false}
                onClick={onClick}
                world={world}
            />
        </div>
    );
}
