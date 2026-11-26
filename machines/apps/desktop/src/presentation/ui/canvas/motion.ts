import { useSyncExternalStore } from 'react';

// value: the prototype's motion (`T_NODE`, `T_FACE`, the camera's and the exits' durations in `renderCanvas` and
// value: `setCam` of wiki-proto/spwn-wiki-proto-r1.html), in ms; the node curve is the --spring token.
export const MOTION = {
    bornMost: 14,
    bornStagger: 14,
    bornScale: 0.3,
    camera: 600,
    cameraCurve: 'cubic-bezier(0.2, 1.04, 0.3, 1)',
    exit: 300,
    exitScale: 0.2,
    face: 260,
    faceDelay: 70,
    ghost: 170,
    guideIn: 300,
    node: 540,
} as const;

const REDUCED = '(prefers-reduced-motion: reduce)';

function subscribe(onChange: () => void): () => void {
    const query = globalThis.matchMedia(REDUCED);
    query.addEventListener('change', onChange);
    return () => {
        query.removeEventListener('change', onChange);
    };
}

function moves(): boolean {
    return !globalThis.matchMedia(REDUCED).matches;
}

// value: whether the canvas animates: false under prefers-reduced-motion, where every change jumps to its end.
export function useMotion(): boolean {
    return useSyncExternalStore(subscribe, moves);
}

export type Curves = { spring: string; ease: string };

// hazard: a page that loads no tokens (a component spec) reads an empty curve, which WAAPI refuses: it runs linear.
function curveOf(style: CSSStyleDeclaration, token: string): string {
    const value = style.getPropertyValue(token).trim();
    return value === '' ? 'linear' : value;
}

// value: the token curves as WAAPI reads them, from the element's computed style, once per canvas.
export function curvesOf(element: Element): Curves {
    const style = getComputedStyle(element);
    return { ease: curveOf(style, '--ease'), spring: curveOf(style, '--spring') };
}
