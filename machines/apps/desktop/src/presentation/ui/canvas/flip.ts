import type { Camera } from './camera';
import type { CanvasGraph, Placed, Rect } from './graph';
import { MOTION } from './motion';
import type { Curves } from './motion';

export type Drawn = { rect: Placed; element: HTMLElement };

export type Scene = {
    graph: CanvasGraph;
    before: ReadonlyMap<string, Placed>;
    after: ReadonlyMap<string, Placed>;
    curves: Curves;
};

function translate({ x, y }: { x: number; y: number }): string {
    return `translate(${String(x)}px, ${String(y)}px)`;
}

// value: a node's place, written as the custom properties its module's CSS reads, and nothing else of its style.
export function seat(element: HTMLElement, rect: Rect): void {
    element.style.setProperty('--x', `${String(rect.x)}px`);
    element.style.setProperty('--y', `${String(rect.y)}px`);
    element.style.setProperty('--w', `${String(rect.w)}px`);
    element.style.setProperty('--h', `${String(rect.h)}px`);
}

function nearest(
    graph: CanvasGraph,
    id: string,
    placed: ReadonlyMap<string, Placed>,
): Placed | undefined {
    let at = graph.nodes.get(id)?.parent ?? null;
    while (at !== null) {
        const rect = placed.get(at);
        if (rect !== undefined) {
            return rect;
        }
        at = graph.nodes.get(at)?.parent ?? null;
    }
    return undefined;
}

function part(element: HTMLElement, name: string): HTMLElement | null {
    return element.querySelector<HTMLElement>(`:scope > [data-part="${name}"]`);
}

// invariant: a node that stays moves from its old place by a transform, its ground scales from its old size, and a
// invariant: face that changed fades in over its ghost: transform and opacity only, from the rects the layout gave,
// invariant: so no frame reads the DOM.
function moved(scene: Scene, { element, rect }: Drawn, old: Placed, changed: boolean): void {
    const { node } = MOTION;
    if (old.x !== rect.x || old.y !== rect.y) {
        element.animate([{ transform: translate(old) }, { transform: translate(rect) }], {
            duration: node,
            easing: scene.curves.spring,
        });
    }
    const ground = part(element, 'ground');
    if (ground !== null && (old.w !== rect.w || old.h !== rect.h)) {
        ground.animate(
            [
                { transform: `scale(${String(old.w / rect.w)}, ${String(old.h / rect.h)})` },
                { transform: 'none' },
            ],
            { duration: node, easing: scene.curves.spring },
        );
    }
    const ghost = part(element, 'ghost');
    if (changed && ghost !== null) {
        ghost.style.setProperty('--gw', `${String(old.w)}px`);
        ghost.style.setProperty('--gh', `${String(old.h)}px`);
        ghost.animate([{ opacity: 1 }, { opacity: 0 }], {
            duration: MOTION.ghost,
            easing: scene.curves.ease,
            fill: 'forwards',
        });
    }
    const body = part(element, 'body');
    if (changed && body !== null) {
        body.animate([{ opacity: 0 }, { opacity: 1 }], {
            delay: MOTION.faceDelay,
            duration: MOTION.face,
            easing: scene.curves.ease,
            fill: 'backwards',
        });
    }
}

function born(scene: Scene, { element, rect }: Drawn, order: number): void {
    const from = nearest(scene.graph, element.dataset.node ?? '', scene.before);
    const start =
        from === undefined
            ? `${translate({ x: rect.x, y: rect.y + 14 })} scale(0.92)`
            : `${translate({
                  x: from.x + from.w / 2 - rect.w * (MOTION.bornScale / 2),
                  y: from.y + from.h / 2 - rect.h * (MOTION.bornScale / 2),
              })} scale(${String(MOTION.bornScale)})`;
    element.animate(
        [
            { opacity: 0, transform: start },
            { opacity: 1, transform: translate(rect) },
        ],
        {
            delay: Math.min(order, MOTION.bornMost) * MOTION.bornStagger,
            duration: MOTION.node,
            easing: scene.curves.spring,
            fill: 'backwards',
        },
    );
}

// value: a node the new layout drops shrinks into its nearest ancestor still drawn, and fades.
export function leave(scene: Scene, { element, rect }: Drawn): void {
    const into = nearest(scene.graph, element.dataset.node ?? '', scene.after);
    const end =
        into === undefined
            ? `${translate(rect)} scale(0.9)`
            : `${translate({
                  x: into.x + Math.min(into.w, 40) / 2,
                  y: into.y + Math.min(into.h, 30) / 2,
              })} scale(${String(MOTION.exitScale)})`;
    element.animate(
        [
            { opacity: 1, transform: translate(rect) },
            { opacity: 0, transform: end },
        ],
        { duration: MOTION.exit, easing: scene.curves.ease, fill: 'forwards' },
    );
}

// value: every node drawn after a layout change, placed, then moved, born or faded by its old rect; the born
// value: stagger counts in depth order, as the prototype's renderCanvas does.
export function flip(scene: Scene, drawn: readonly Drawn[]): void {
    let order = 0;
    for (const one of drawn) {
        seat(one.element, one.rect);
        const old = scene.before.get(one.element.dataset.node ?? '');
        if (old === undefined) {
            born(scene, one, order);
            order += 1;
        } else {
            moved(scene, one, old, old.face !== one.rect.face);
        }
    }
}

// invariant: the world's own transform, never an inherited custom property that restyles every node on a pan.
export function place(world: HTMLElement, at: Camera): void {
    world.style.transform = translate(at);
}

// value: the camera carried to `to` from where it shows now, over the prototype's camera curve.
export function carry(world: HTMLElement, from: Camera, to: Camera): Animation | null {
    place(world, to);
    if (Math.abs(from.x - to.x) <= 1 && Math.abs(from.y - to.y) <= 1) {
        return null;
    }
    return world.animate([{ transform: translate(from) }, { transform: translate(to) }], {
        duration: MOTION.camera,
        easing: MOTION.cameraCurve,
    });
}

// value: where the camera shows while it is carried, read once, when a gesture interrupts it.
export function liveCamera(world: HTMLElement): Camera {
    const matrix = new DOMMatrixReadOnly(getComputedStyle(world).transform);
    return { x: matrix.e, y: matrix.f };
}
