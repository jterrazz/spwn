'use client';

import { getWorldName } from '../domain/model';
import type { World } from '../domain/model';
import { useIsMobile } from '../hooks/use-mobile';
import { PLACEHOLDER_ID, planetColours } from './planet/geometry';
import { useGlobe } from './planet/use-globe';

const EASE = 'cubic-bezier(0.16, 1, 0.3, 1)';
const AGENT_GLOW = {
    background:
        'radial-gradient(circle, rgba(255,255,255,0.95) 0%, rgba(255,255,255,0.35) 20%, rgba(255,255,255,0) 65%)',
    borderRadius: '50%',
    boxShadow: '0 0 12px 5px rgba(255,255,255,0.5), 0 0 28px 10px rgba(255,255,255,0.15)',
    height: 28,
    opacity: 0,
    transform: 'translate(-50%, -50%)',
    transition: 'opacity 0.3s',
    width: 28,
};

interface PlanetProps {
    world: World;
    index: number;
    onClick: () => void;
    isSelected: boolean;
    compact?: boolean;
    hideLabels?: boolean;
}

function scaleOf(selected: boolean, mobile: boolean, compact: boolean): number {
    if (mobile) {
        return selected ? 1.3 : 0.7;
    }
    if (compact) {
        return selected ? 1.5 : 1.15;
    }
    return selected ? 1.8 : 0.85;
}

function PlusMark() {
    return (
        <svg
            aria-hidden="true"
            className="pointer-events-none absolute inset-0 z-20 m-auto text-white/90"
            fill="none"
            height="28"
            viewBox="0 0 20 20"
            width="28"
        >
            <path
                d="M10 4.5V15.5M4.5 10H15.5"
                stroke="currentColor"
                strokeLinecap="round"
                strokeWidth="2.4"
            />
        </svg>
    );
}

function Label({ name, selected }: { name: string; selected: boolean }) {
    return (
        <span
            className="font-heading pointer-events-none absolute left-1/2 -translate-x-1/2 text-center tracking-wider whitespace-nowrap"
            style={{
                bottom: `calc(100% + ${selected ? 40 : 12}px)`,
                color: selected ? 'rgba(255,255,255,0.95)' : 'rgba(255,255,255,0.5)',
                fontSize: selected ? '1.15rem' : '0.8rem',
                letterSpacing: selected ? '0.12em' : '0.05em',
                opacity: selected ? 1 : 0.6,
                transition: `bottom 0.9s ${EASE}, opacity 0.7s ease-out, color 0.7s ease-out, font-size 0.7s ease-out, letter-spacing 0.7s ease-out`,
            }}
        >
            {name}
        </span>
    );
}

function AgentGlows({
    world,
    glows,
}: {
    world: World;
    glows: React.RefObject<Map<string, HTMLSpanElement>>;
}) {
    return (
        <>
            {world.agents.map((agent) => (
                <span
                    className="pointer-events-none absolute"
                    key={`glow-${agent.name}`}
                    ref={(element) => {
                        if (element === null) {
                            glows.current.delete(agent.name);
                        } else {
                            glows.current.set(agent.name, element);
                        }
                    }}
                    style={AGENT_GLOW}
                />
            ))}
        </>
    );
}

/** A world drawn as a spinning planet: its colour its id's, its agents glowing on its surface, its name above it. */
export function Planet({
    world,
    index,
    onClick,
    isSelected,
    compact = false,
    hideLabels = false,
}: PlanetProps) {
    const mobile = useIsMobile();
    const size = compact || mobile ? 140 : 200;
    const { canvasRef, glows, wrapperRef } = useGlobe({
        compact,
        index,
        selected: isSelected,
        size,
        world,
    });
    const { hue, saturation } = planetColours(world.id, world.status);
    const halo = isSelected
        ? `brightness(1.2) drop-shadow(0 0 28px hsl(${hue}, ${saturation}%, 62%))`
        : `brightness(1) drop-shadow(0 0 12px hsla(${hue}, ${saturation}%, 60%, 0.45))`;

    return (
        <button
            className="relative flex cursor-pointer items-center justify-center will-change-transform focus:outline-none"
            onClick={onClick}
            style={{ height: size, width: size }}
            type="button"
        >
            <span
                className="block will-change-[transform,filter]"
                ref={wrapperRef}
                style={{
                    filter: halo,
                    height: size,
                    transform: `scale(${scaleOf(isSelected, mobile, compact)})`,
                    transition: `transform 0.9s ${EASE}, filter 0.9s ${EASE}`,
                    width: size,
                }}
            >
                <canvas ref={canvasRef} style={{ height: size, width: size }} />
                {world.id === PLACEHOLDER_ID && <PlusMark />}
                <AgentGlows glows={glows} world={world} />
            </span>
            {!hideLabels && <Label name={getWorldName(world)} selected={isSelected} />}
        </button>
    );
}
