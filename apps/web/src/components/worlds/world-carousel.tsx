'use client';

import type { CSSProperties, RefObject } from 'react';

import type { World } from '../../domain/model';
import { NewWorldCard } from '../new-world-card';
import { Planet as PlanetGlobe } from '../planet';
import { useCarousel } from './use-carousel';
import { WorldPanel } from './world-panel';

const SETTLE = 'transform 0.8s cubic-bezier(0.16, 1, 0.3, 1)';
const RECEDE =
    'opacity 0.7s ease-out, transform 0.7s ease-out, filter 0.7s ease-out, margin 0.9s cubic-bezier(0.16, 1, 0.3, 1)';

interface WorldCarouselProps {
    worlds: World[];
    selected: null | number;
    onSelect: (index: null | number) => void;
    onEnter: (world: World) => void;
    onShutdown: (world: World) => void;
    onSpawn: () => void;
}

interface PlanetRowProps extends Pick<
    WorldCarouselProps,
    'onSelect' | 'onSpawn' | 'selected' | 'worlds'
> {
    planets: RefObject<(HTMLDivElement | null)[]>;
    wasDrag: () => boolean;
}

function planetStyle(active: boolean, receding: boolean): CSSProperties {
    return {
        filter: receding ? 'blur(1px)' : 'blur(0px)',
        margin: active ? '0 36px' : '0',
        opacity: receding ? 0.35 : 1,
        transform: receding ? 'scale(0.9)' : 'scale(1)',
        transition: RECEDE,
    };
}

function PlanetRow({ worlds, selected, onSelect, onSpawn, planets, wasDrag }: PlanetRowProps) {
    return (
        <>
            {worlds.map((world, index) => (
                <div
                    className="flex shrink-0 cursor-pointer flex-col items-center"
                    key={world.id}
                    ref={(element) => {
                        planets.current[index] = element;
                    }}
                    style={planetStyle(selected === index, selected !== null && selected !== index)}
                >
                    <PlanetGlobe
                        compact
                        index={index}
                        isSelected={selected === index}
                        onClick={() => {
                            if (!wasDrag()) {
                                onSelect(selected === index ? null : index);
                            }
                        }}
                        world={world}
                    />
                </div>
            ))}
            <NewWorldCard
                onClick={onSpawn}
                opacity={selected === null ? 0.5 : 0.2}
                scale={selected === null ? 1 : 0.85}
                tint="creating"
            />
        </>
    );
}

interface TrackProps extends PlanetRowProps {
    dragging: boolean;
    offset: number;
    startDrag: (x: number) => void;
    trackRef: RefObject<HTMLDivElement | null>;
}

function Track({ dragging, offset, startDrag, trackRef, ...row }: TrackProps) {
    return (
        <div className="flex h-full items-center pb-24">
            <div
                className="flex items-center gap-10 will-change-transform select-none"
                onMouseDown={(event) => {
                    event.preventDefault();
                    startDrag(event.clientX);
                }}
                onTouchStart={(event) => {
                    const touch = event.touches.item(0);
                    if (touch !== null) {
                        startDrag(touch.clientX);
                    }
                }}
                ref={trackRef}
                role="presentation"
                style={{
                    cursor: dragging ? 'grabbing' : 'grab',
                    transform: `translateX(${offset}px)`,
                    transition: dragging ? 'none' : SETTLE,
                }}
            >
                <PlanetRow {...row} />
            </div>
        </div>
    );
}

/** The worlds as a row of planets the visitor drags or walks with the arrow keys; selecting one opens its panel. */
export function WorldCarousel({
    worlds,
    selected,
    onSelect,
    onEnter,
    onShutdown,
    onSpawn,
}: WorldCarouselProps) {
    const { panelRef, ...carousel } = useCarousel(worlds.length, selected);
    const focused = selected === null ? undefined : worlds[selected];

    return (
        <div className="relative -mx-6 min-h-[320px] flex-1 overflow-hidden md:-mx-8">
            <Track
                {...carousel}
                onSelect={onSelect}
                onSpawn={onSpawn}
                selected={selected}
                worlds={worlds}
            />
            {focused !== undefined && (
                <WorldPanel
                    onClose={() => {
                        onSelect(null);
                    }}
                    onEnter={() => {
                        onEnter(focused);
                    }}
                    onShutdown={() => {
                        onShutdown(focused);
                    }}
                    panelRef={panelRef}
                    world={focused}
                />
            )}
        </div>
    );
}
