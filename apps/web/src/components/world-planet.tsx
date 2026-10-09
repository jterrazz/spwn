import type { World } from '../domain/model';
import { worldHue } from '../domain/world-hue';

interface WorldPlanetProps {
    world: Pick<World, 'id' | 'status'>;
    size?: 'lg' | 'md' | 'sm';
    className?: string;
}

/** A small gradient sphere representing a world. Color is derived from the world's id hash so the same world is visually identifiable everywhere; saturation reflects activity (active worlds saturate, idle/stopped ones desaturate toward grey). Shared across sidebar, page headers, and any other surface where a world gets displayed - keeping rendering identical by construction. */
export function WorldPlanet({ world, size = 'md', className = '' }: WorldPlanetProps) {
    const hue = worldHue(world.id);
    const isActive = world.status === 'running' || world.status === 'creating';
    const sat = isActive ? 70 : 15;

    const gradient = `radial-gradient(circle at 32% 30%, hsl(${hue} ${sat}% 78%), hsl(${hue} ${sat}% 48%) 55%, hsl(${hue} ${sat}% 22%))`;

    if (size === 'sm') {
        return (
            <span
                className={`block h-3 w-3 shrink-0 rounded-full ${className}`}
                style={{
                    background: gradient,
                    boxShadow: 'inset 0 -1px 1px rgba(0,0,0,0.35)',
                }}
            />
        );
    }

    const sizeClass = size === 'lg' ? 'w-9 h-9' : 'w-6 h-6';
    const haloInset = size === 'lg' ? 'inset-[-10px]' : 'inset-[-8px]';
    const haloBlur = size === 'lg' ? 'blur-xl' : 'blur-lg';
    const ringWidth = size === 'lg' ? 2 : 1.5;

    return (
        <span
            className={`relative shrink-0 ${sizeClass} flex items-center justify-center overflow-visible ${className}`}
        >
            <span
                className={`absolute ${haloInset} rounded-full ${haloBlur} pointer-events-none`}
                style={{ background: `hsl(${hue} ${sat}% 60% / 0.25)` }}
            />
            <span
                className={`relative block ${sizeClass} rounded-full`}
                style={{
                    background: gradient,
                    boxShadow: `0 0 0 ${ringWidth}px hsl(${hue} ${sat}% 72% / 0.9), inset 0 -1px 2px rgba(0,0,0,0.35)`,
                }}
            />
        </span>
    );
}
