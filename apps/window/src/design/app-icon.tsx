import { createElement } from 'react';

import { markOf } from './app-marks';
import type { Shape } from './app-marks';
import { Icon } from './icon';

import styles from './app-icon.module.css';

function Shapes({ shapes }: { shapes: readonly Shape[] }) {
    return shapes.map((shape) =>
        createElement(
            shape.tag,
            { ...shape.attributes, key: shape.key },
            shape.children.length > 0 ? <Shapes shapes={shape.children} /> : undefined,
        ),
    );
}

function Drawn({ name, kind }: { name: string; kind: string | undefined }) {
    const mark = markOf(name, kind);
    switch (mark.kind) {
        case 'glyph': {
            return <Icon name={mark.glyph} />;
        }
        case 'drawn': {
            return (
                <svg aria-hidden="true" className={styles.mark} viewBox="0 0 20 20">
                    <Shapes shapes={mark.shapes} />
                </svg>
            );
        }
        case 'monogram': {
            return <span className={styles.letters}>{mark.letters}</span>;
        }
    }
}

export function AppIcon({ name, kind }: { name: string; kind?: string | undefined }) {
    return (
        <span aria-hidden="true" className={styles.icon}>
            <Drawn kind={kind} name={name} />
        </span>
    );
}
