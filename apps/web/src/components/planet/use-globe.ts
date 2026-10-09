'use client';

import createGlobe from 'cobe';
import type { Globe } from 'cobe';
import { useEffect, useRef } from 'react';

import type { World } from '../../domain/model';
import { agentLocation, continents, idleDiffuse, planetColours, startingSpin } from './geometry';

interface Spin {
    globe: Globe;
    phi: number;
    frame: number;
    restTheta: number;
    restDiffuse: number;
}

interface GlobeOptions {
    world: World;
    index: number;
    size: number;
    compact: boolean;
    selected: boolean;
}

interface Drawing {
    id: string;
    status: string;
    agents: string[];
    index: number;
    size: number;
}

// value: false on a frame a compact, idle planet skips, which moves no marker either
function step(spin: Spin, selected: boolean, compact: boolean): boolean {
    spin.frame += 1;
    if (compact && !selected && spin.frame % 3 !== 0) {
        return false;
    }
    if (selected) {
        spin.phi += 0.005;
        spin.globe.update({
            diffuse: 1.6 + Math.sin(spin.frame * 0.015) * 0.3,
            phi: spin.phi,
            scale: 1 + Math.sin(spin.frame * 0.012) * 0.015,
            theta: spin.restTheta + Math.sin(spin.frame * 0.008) * 0.12,
        });
        return true;
    }
    spin.phi += compact ? 0.006 : 0.002;
    spin.globe.update({
        diffuse: spin.restDiffuse,
        phi: spin.phi,
        scale: 1,
        theta: spin.restTheta,
    });
    return true;
}

// outside: cobe draws each marker as an anchor element carrying its position and visibility as CSS variables; a glow follows its anchor
function followMarkers(wrapper: HTMLSpanElement | null, glows: Map<string, HTMLSpanElement>): void {
    if (wrapper === null) {
        return;
    }
    for (const [name, glow] of glows) {
        const anchor = wrapper.querySelector(`[style*="--cobe-marker-${name}"]`);
        if (anchor instanceof HTMLElement) {
            glow.style.left = anchor.style.left;
            glow.style.top = anchor.style.top;
            glow.style.opacity =
                getComputedStyle(anchor).getPropertyValue(`--cobe-marker-visible-${name}`) === ''
                    ? '0'
                    : '1';
        }
    }
}

function drawGlobe(
    canvas: HTMLCanvasElement,
    { id, status, agents: names, index, size }: Drawing,
): Globe {
    const colours = planetColours(id, status);
    const agents = names.map((name) => ({ location: agentLocation(name), size: 0.03 }));
    return createGlobe(canvas, {
        baseColor: colours.base,
        dark: 1,
        devicePixelRatio: 2,
        diffuse: idleDiffuse(id),
        // outside: cobe renders an inside-out halo when glowColor is not darker than baseColor; the halo comes from the CSS drop-shadow instead
        glowColor: colours.glow,
        height: size * 2,
        mapBaseBrightness: 0,
        mapBrightness: 0,
        mapSamples: 0,
        markerColor: colours.marker,
        markerElevation: 0,
        markers: [...continents(id, Math.round(size * 5)), ...agents],
        phi: startingSpin(id),
        theta: 0.15 + index * 0.3,
        width: size * 2,
    });
}

/** Draws a world's planet on the canvas and keeps it spinning — faster and breathing while it is selected — with its agents' glows following their markers. */
export function useGlobe(options: GlobeOptions) {
    const canvasRef = useRef<HTMLCanvasElement>(null);
    const wrapperRef = useRef<HTMLSpanElement>(null);
    const glows = useRef<Map<string, HTMLSpanElement>>(new Map());
    const selected = useRef(options.selected);
    const { index, size, compact } = options;
    const { id, status } = options.world;
    const agentNames = options.world.agents.map((agent) => agent.name).join('\n');

    useEffect(() => {
        selected.current = options.selected;
    }, [options.selected]);

    useEffect(() => {
        if (canvasRef.current === null) {
            return;
        }
        const agents = agentNames === '' ? [] : agentNames.split('\n');
        const globe = drawGlobe(canvasRef.current, { agents, id, index, size, status });
        const spin: Spin = {
            frame: 0,
            globe,
            phi: startingSpin(id),
            restDiffuse: idleDiffuse(id),
            restTheta: 0.15 + index * 0.3,
        };
        let frame = 0;
        const animate = () => {
            if (step(spin, selected.current, compact)) {
                followMarkers(wrapperRef.current, glows.current);
            }
            frame = requestAnimationFrame(animate);
        };
        frame = requestAnimationFrame(animate);
        return () => {
            cancelAnimationFrame(frame);
            globe.destroy();
        };
    }, [id, status, agentNames, index, size, compact]);

    return { canvasRef, glows, wrapperRef };
}
