import { positiveHash, worldHue } from '../../domain/world-hue';

export const PLACEHOLDER_ID = 'w-new-00000';

const STATUS_SATURATION: Record<string, number> = {
    creating: 55,
    error: 85,
    idle: 40,
    running: 55,
    stopped: 15,
};

type Rgb = [number, number, number];

interface Marker {
    location: [number, number];
    size: number;
}

function hslToRgb01(hue: number, saturation: number, lightness: number): Rgb {
    const s = saturation / 100;
    const l = lightness / 100;
    const k = (n: number) => (n + hue / 30) % 12;
    const a = s * Math.min(l, 1 - l);
    const f = (n: number) => l - a * Math.max(-1, Math.min(k(n) - 3, 9 - k(n), 1));
    return [f(0), f(8), f(4)];
}

/** The colours a world's planet is drawn in: its hue from its id, its saturation from how alive it is. */
export function planetColours(worldId: string, status: string) {
    const hue = status === 'error' ? 0 : worldHue(worldId);
    const saturation = worldId === PLACEHOLDER_ID ? 0 : (STATUS_SATURATION[status] ?? 20);
    return {
        base: hslToRgb01(hue, saturation, 30),
        glow: hslToRgb01(hue, saturation, 12),
        hue,
        marker: hslToRgb01(hue, saturation, 92),
        saturation,
    };
}

export function idleDiffuse(worldId: string): number {
    return 1.3 + (positiveHash(worldId) % 10) * 0.05;
}

export function startingSpin(worldId: string): number {
    return (positiveHash(worldId) % 628) / 100;
}

/** Where an agent's marker sits on its planet: a fixed point derived from its name. */
export function agentLocation(name: string): [number, number] {
    const hash = positiveHash(name);
    const lat = ((hash % 1000) / 1000) * 140 - 70;
    const lng = ((Math.floor(hash / 1024) % 1000) / 1000) * 360 - 180;
    return [lat, lng];
}

function seeded(seed: number) {
    let state = seed;
    return () => {
        state = (state * 16_807) % 2_147_483_647;
        return state / 2_147_483_647;
    };
}

/** The dots of a planet's continents, the same for the same world id. */
export function continents(worldId: string, count: number): Marker[] {
    const random = seeded(positiveHash(worldId));
    const centers = Array.from({ length: 5 + Math.floor(random() * 5) }, (): [number, number] => [
        random() * 140 - 70,
        random() * 360 - 180,
    ]);
    const perContinent = Math.floor(count / centers.length);
    return centers.flatMap(([lat, lng]) => {
        const spread = 25 + random() * 35;
        return Array.from({ length: perContinent }, (): Marker => {
            const angle = random() * Math.PI * 2;
            const distance = ((random() + random() + random()) / 3) * spread;
            const markerLng = lng + Math.sin(angle) * distance;
            return {
                location: [
                    Math.max(-85, Math.min(85, lat + Math.cos(angle) * distance)),
                    ((markerLng + 180) % 360) - 180,
                ],
                size: 0.006 + random() * 0.012,
            };
        });
    });
}
