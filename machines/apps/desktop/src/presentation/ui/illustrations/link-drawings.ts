import { fixed, Iso, projected } from './iso';
import type { Drawing, Spot } from './iso';
import { badge, dash, diamond, rays, sign, sparks } from './iso-marks';

export type MacsLink = 'joined' | 'linking' | 'searching';

function searching(iso: Iso): void {
    const [x, y] = projected([72, 0, 30]);
    const fades = ['none', 'more', 'most'] as const;
    for (const [at, r] of [10, 16, 22].entries()) {
        const fade = fades[at];
        const d = `M${String(fixed(x - r))},${String(fixed(y))} A${String(r)},${String(fixed(r * 0.7))} 0 0 1 ${String(fixed(x + r))},${String(fixed(y))}`;
        iso.path(d, fade === 'none' || fade === undefined ? {} : { fade });
    }
    iso.reached([x - 22, y - 16]);
}

export function macs(link: MacsLink): Drawing {
    const iso = new Iso();
    iso.shadow(-104, -28, [64, 56], 9);
    iso.box([-104, -28, 0], [64, 56, 14]);
    sign(iso, [-72, 2, 14], 0.62);
    const wire: Spot[] = [];
    for (let step = 0; step <= 24; step += 1) {
        const t = step / 24;
        wire.push([-40 + 80 * t, 12, 7 * (1 - Math.sin(Math.PI * t)) + 0.5]);
    }
    iso.line(
        wire,
        link === 'searching'
            ? { dash: 'long', ink: 'dim' }
            : { ink: link === 'linking' ? 'accent' : 'ink' },
    );
    iso.shadow(40, -28, [64, 56], 9);
    iso.box([40, -28, 0], [64, 56, 14]);
    iso.dot(projected([92, 28, 7]), 2.4, link === 'searching' ? 'dim' : 'accent');
    if (link === 'searching') {
        searching(iso);
    }
    if (link === 'joined') {
        badge(iso, [72, 0, 44], 11, 'check');
        rays(iso, [72, 0, 44], [16, 22], [-150, -110, -70, -30]);
    }
    if (link === 'linking') {
        diamond(iso, [0, -40, 40], 4);
        dash(iso, [-10, 46, 26], 150, 8);
    }
    return iso.drawing();
}

export function key(): Drawing {
    const iso = new Iso();
    iso.shadow(-62, -42, [124, 84], 10);
    iso.box([-62, -42, 0], [124, 84, 8]);
    iso.cylinder([-30, 0, 8], [17, 5], { ink: 'accent', paint: 'litTop', side: 'litLeft' });
    iso.ellipse([-30, 0, 13], 7, { ink: 'accent', paint: 'shadow' });
    iso.box([-14, -4, 8], [58, 8, 5], { lit: true });
    iso.box([30, 4, 8], [6, 9, 5], { lit: true });
    iso.box([40, 4, 8], [5, 6, 5], { lit: true });
    sparks(iso, 70, 46);
    return iso.drawing();
}

export function done(): Drawing {
    const iso = new Iso();
    iso.shadow(-46, -46, [92, 92], 10);
    iso.box([-46, -46, 0], [92, 92, 14]);
    iso.box([-28, -28, 14], [56, 56, 5]);
    badge(iso, [0, 0, 60], 13, 'check');
    rays(iso, [0, 0, 60], [19, 26], [-160, -125, -90, -55, -20]);
    sparks(iso, 74, 50);
    return iso.drawing();
}
