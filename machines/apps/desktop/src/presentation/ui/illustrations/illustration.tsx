import { boot, bootFailed, build, engine, settings, touch } from './drawings';
import type { Drawing, Shape } from './iso';
import { done, key, macs } from './link-drawings';

import styles from './illustration.module.css';

// value: the drawings a guide's step can show, one per moment of an act; the engine names one, the window draws it.
export const ILLUSTRATIONS = [
    'install-engine',
    'fetch-image',
    'build-1',
    'build-2',
    'build-3',
    'build-4',
    'build-failed',
    'boot',
    'boot-failed',
    'apply-file',
    'system-settings',
    'touch-id',
    'mac-search',
    'mac-link',
    'mac-joined',
    'device-key',
    'done',
] as const;

export type IllustrationName = (typeof ILLUSTRATIONS)[number];

const DRAWN: Readonly<Record<IllustrationName, Drawing>> = {
    'apply-file': boot(true),
    boot: boot(false),
    'boot-failed': bootFailed(),
    'build-1': build(1, false),
    'build-2': build(2, false),
    'build-3': build(3, false),
    'build-4': build(4, false),
    'build-failed': build(3, true),
    'device-key': key(),
    done: done(),
    'fetch-image': engine(true),
    'install-engine': engine(false),
    'mac-joined': macs('joined'),
    'mac-link': macs('linking'),
    'mac-search': macs('searching'),
    'system-settings': settings(),
    'touch-id': touch(),
};

export function drawingOf(name: IllustrationName): Drawing {
    return DRAWN[name];
}

type Marked = {
    'data-dash'?: string | undefined;
    'data-fade'?: string | undefined;
    'data-glyph'?: string | undefined;
    'data-ink': string;
    'data-paint': string;
};

function markedOf({ look }: Shape): Marked {
    return {
        'data-dash': look.dash,
        'data-fade': look.fade,
        'data-glyph': look.glyph === true ? '' : undefined,
        'data-ink': look.ink ?? 'ink',
        'data-paint': look.paint ?? 'none',
    };
}

function Drawn({ shape }: { shape: Shape }) {
    const marked = markedOf(shape);
    const { geometry } = shape;
    switch (geometry.tag) {
        case 'circle': {
            return <circle cx={geometry.cx} cy={geometry.cy} r={geometry.r} {...marked} />;
        }
        case 'ellipse': {
            return (
                <ellipse
                    cx={geometry.cx}
                    cy={geometry.cy}
                    rx={geometry.rx}
                    ry={geometry.ry}
                    {...marked}
                />
            );
        }
        case 'path': {
            return <path d={geometry.d} {...marked} />;
        }
        case 'polygon': {
            return <polygon points={geometry.points} {...marked} />;
        }
        case 'polyline': {
            return <polyline points={geometry.points} {...marked} />;
        }
    }
}

export function Illustration({ name }: { name: IllustrationName }) {
    const drawing = DRAWN[name];
    return (
        <svg
            aria-hidden="true"
            className={styles.art}
            data-illustration={name}
            viewBox={drawing.viewBox}
        >
            {drawing.shapes.map((shape) => (
                <Drawn key={shape.id} shape={shape} />
            ))}
        </svg>
    );
}
