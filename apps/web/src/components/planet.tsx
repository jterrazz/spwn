'use client';

import createGlobe from 'cobe';
import type { Globe } from 'cobe';
import { useEffect, useRef, useState } from 'react';

import { getWorldName } from '../domain/model';
import type { World } from '../domain/model';

interface PlanetProps {
    world: World;
    index: number;
    onClick: () => void;
    onEnter?: () => void;
    isSelected: boolean;
    compact?: boolean;
    hideLabels?: boolean;
}

const STATUS_SAT: Record<string, number> = {
    running: 55,
    creating: 55,
    idle: 40,
    error: 85,
    stopped: 15,
};

// invariant: the sidebar draws the same hue for the same world id, so a world reads as one planet everywhere
function hashHue(id: null | string | undefined): number {
    if (id === null || id === undefined || id === '') {
        return 0;
    }
    let h = 0;
    for (let i = 0; i < id.length; i++) {
        h = (h * 31 + id.charCodeAt(i)) >>> 0;
    }
    return h % 360;
}

function hslToRgb01(h: number, s: number, l: number): [number, number, number] {
    const sN = s / 100;
    const lN = l / 100;
    const k = (n: number) => (n + h / 30) % 12;
    const a = sN * Math.min(lN, 1 - lN);
    const f = (n: number) => lN - a * Math.max(-1, Math.min(k(n) - 3, 9 - k(n), 1));
    return [f(0), f(8), f(4)];
}

function getPlanetConfig(worldId: string, status: string) {
    const isPlaceholder = worldId === 'w-new-00000';
    const hue = status === 'error' ? 0 : hashHue(worldId);
    const sat = isPlaceholder ? 0 : (STATUS_SAT[status] ?? 20);
    return {
        hue,
        sat,
        base: hslToRgb01(hue, sat, 30),
        glow: hslToRgb01(hue, sat, 12),
        marker: hslToRgb01(hue, sat, 92),
        haloCss: `hsl(${hue}, ${sat}%, 62%)`,
    };
}

function hashCode(s: null | string | undefined): number {
    if (s === null || s === undefined || s === '') {
        return 0;
    }
    let h = 0;
    for (let i = 0; i < s.length; i++) {
        h = (Math.imul(31, h) + s.charCodeAt(i)) | 0;
    }
    return Math.abs(h);
}

function stringToLocation(name: string): [number, number] {
    const h = hashCode(name);
    const lat = ((h % 1000) / 1000) * 140 - 70;
    const lng = (((h >> 10) % 1000) / 1000) * 360 - 180;
    return [lat, lng];
}

function seeded(seed: number) {
    let s = seed;
    return () => {
        s = (s * 16_807 + 0) % 2_147_483_647;
        return s / 2_147_483_647;
    };
}

function generateContinents(
    worldId: string,
    count: number,
): { location: [number, number]; size: number }[] {
    const rng = seeded(hashCode(worldId));
    const markers: { location: [number, number]; size: number }[] = [];

    const numContinents = 5 + Math.floor(rng() * 5);
    const centers: [number, number][] = [];
    for (let i = 0; i < numContinents; i++) {
        centers.push([rng() * 140 - 70, rng() * 360 - 180]);
    }

    const dotsPerContinent = Math.floor(count / numContinents);
    for (const [cLat, cLng] of centers) {
        const spread = 25 + rng() * 35;
        for (let j = 0; j < dotsPerContinent; j++) {
            const angle = rng() * Math.PI * 2;
            const dist = ((rng() + rng() + rng()) / 3) * spread;
            const lat = cLat + Math.cos(angle) * dist;
            const lng = cLng + Math.sin(angle) * dist;
            markers.push({
                location: [Math.max(-85, Math.min(85, lat)), ((lng + 180) % 360) - 180],
                size: 0.006 + rng() * 0.012,
            });
        }
    }

    return markers;
}

export function Planet({
    world,
    index,
    onClick,
    onEnter: _onEnter,
    isSelected,
    compact,
    hideLabels,
}: PlanetProps) {
    const canvasRef = useRef<HTMLCanvasElement>(null);
    const globeRef = useRef<Globe | null>(null);
    const phiRef = useRef((hashCode(world.id) % 628) / 100);
    const config = getPlanetConfig(world.id, world.status);
    const name = getWorldName(world);
    const [isMobile, setIsMobile] = useState(false);

    useEffect(() => {
        const check = () => {
            setIsMobile(window.innerWidth < 768);
        };
        check();
        window.addEventListener('resize', check);
        return () => {
            window.removeEventListener('resize', check);
        };
    }, []);

    const size: number = compact === true || isMobile ? 140 : 200;

    const wrapperRef = useRef<HTMLDivElement>(null);
    const glowRefs = useRef<Map<string, HTMLDivElement>>(new Map());
    const selectedRef = useRef(isSelected);
    selectedRef.current = isSelected;

    useEffect(() => {
        if (!canvasRef.current) {
            return;
        }

        const dotCount = Math.round(size * 5);
        const continentMarkers = generateContinents(world.id, dotCount);

        const agentMarkers = (world.agents ?? []).map((a) => ({
            location: stringToLocation(a.name),
            size: 0.03,
        }));

        const globe = createGlobe(canvasRef.current, {
            devicePixelRatio: 2,
            width: size * 2,
            height: size * 2,
            phi: phiRef.current,
            theta: 0.15 + index * 0.3,
            dark: 1,
            diffuse: 1.3 + (hashCode(world.id) % 10) * 0.05,
            mapSamples: 0,
            mapBrightness: 0,
            mapBaseBrightness: 0,
            baseColor: config.base,
            markerColor: config.marker,
            // outside: cobe renders an inside-out halo when glowColor is not darker than baseColor; the halo comes from the CSS drop-shadow instead
            glowColor: config.glow,
            markers: [...continentMarkers, ...agentMarkers],
            markerElevation: 0,
        });
        globeRef.current = globe;

        let raf: number;
        let frame = 0;
        const spin = () => {
            frame++;
            const sel = selectedRef.current;

            if (compact === true && !sel && frame % 3 !== 0) {
                raf = requestAnimationFrame(spin);
                return;
            }

            let phiStep: number;
            if (sel) {
                phiStep = 0.005;
            } else if (compact === true) {
                phiStep = 0.006;
            } else {
                phiStep = 0.002;
            }
            phiRef.current += phiStep;

            const theta = sel
                ? 0.15 + index * 0.3 + Math.sin(frame * 0.008) * 0.12
                : 0.15 + index * 0.3;
            const diffuse = sel
                ? 1.6 + Math.sin(frame * 0.015) * 0.3
                : 1.3 + (hashCode(world.id) % 10) * 0.05;
            const scale = sel ? 1 + Math.sin(frame * 0.012) * 0.015 : 1;

            globe.update({ phi: phiRef.current, theta, diffuse, scale });

            const wrapper = wrapperRef.current;
            if (wrapper) {
                (world.agents ?? []).forEach((a) => {
                    const anchor = wrapper.querySelector(
                        `[style*="--cobe-marker-${a.name}"]`,
                    ) as HTMLElement | null;
                    const glow = glowRefs.current.get(a.name);
                    if (anchor && glow) {
                        glow.style.left = anchor.style.left;
                        glow.style.top = anchor.style.top;
                        const vis = getComputedStyle(anchor).getPropertyValue(
                            `--cobe-marker-visible-${a.name}`,
                        );
                        glow.style.opacity = vis ? '1' : '0';
                    }
                });
            }

            raf = requestAnimationFrame(spin);
        };
        raf = requestAnimationFrame(spin);

        return () => {
            cancelAnimationFrame(raf);
            globe.destroy();
        };
    }, [
        world.id,
        world.status,
        world.agents,
        compact,
        size,
        index,
        config.base,
        config.glow,
        config.marker,
    ]);

    const GAP = isSelected ? 40 : 12;

    let globeScale: number;
    if (isSelected) {
        if (isMobile) {
            globeScale = 1.3;
        } else if (compact === true) {
            globeScale = 1.5;
        } else {
            globeScale = 1.8;
        }
    } else if (isMobile) {
        globeScale = 0.7;
    } else if (compact === true) {
        globeScale = 1.15;
    } else {
        globeScale = 0.85;
    }

    return (
        <div
            className="relative flex cursor-pointer items-center justify-center will-change-transform focus:outline-none"
            onClick={onClick}
            role="button"
            style={{ width: size, height: size }}
            tabIndex={0}
        >
            <div
                className="will-change-[transform,filter]"
                ref={wrapperRef}
                style={{
                    width: size,
                    height: size,
                    transform: `scale(${globeScale})`,
                    filter: isSelected
                        ? `brightness(1.2) drop-shadow(0 0 28px hsl(${config.hue}, ${config.sat}%, 62%))`
                        : `brightness(1) drop-shadow(0 0 12px hsla(${config.hue}, ${config.sat}%, 60%, 0.45))`,
                    transition:
                        'transform 0.9s cubic-bezier(0.16, 1, 0.3, 1), filter 0.9s cubic-bezier(0.16, 1, 0.3, 1)',
                }}
            >
                <canvas ref={canvasRef} style={{ width: size, height: size }} />
                {world.id === 'w-new-00000' && (
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
                )}
                {(world.agents ?? []).map((a) => (
                    <div
                        className="pointer-events-none absolute"
                        key={`glow-${a.name}`}
                        ref={(el) => {
                            if (el) {
                                glowRefs.current.set(a.name, el);
                            }
                        }}
                        style={{
                            width: 28,
                            height: 28,
                            transform: 'translate(-50%, -50%)',
                            opacity: 0,
                            background:
                                'radial-gradient(circle, rgba(255,255,255,0.95) 0%, rgba(255,255,255,0.35) 20%, rgba(255,255,255,0) 65%)',
                            boxShadow:
                                '0 0 12px 5px rgba(255,255,255,0.5), 0 0 28px 10px rgba(255,255,255,0.15)',
                            borderRadius: '50%',
                            transition: 'opacity 0.3s',
                        }}
                    />
                ))}
            </div>

            {hideLabels !== true && (
                <p
                    className="font-heading pointer-events-none absolute left-1/2 -translate-x-1/2 text-center tracking-wider whitespace-nowrap"
                    style={{
                        bottom: `calc(100% + ${GAP}px)`,
                        color: isSelected ? 'rgba(255,255,255,0.95)' : 'rgba(255,255,255,0.5)',
                        fontSize: isSelected ? '1.15rem' : '0.8rem',
                        letterSpacing: isSelected ? '0.12em' : '0.05em',
                        opacity: isSelected ? 1 : 0.6,
                        transition:
                            'bottom 0.9s cubic-bezier(0.16, 1, 0.3, 1), opacity 0.7s ease-out, color 0.7s ease-out, font-size 0.7s ease-out, letter-spacing 0.7s ease-out',
                    }}
                >
                    {name}
                </p>
            )}
        </div>
    );
}
