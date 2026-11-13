import type { Identity } from './identity';
import { shownOrWithheld } from './secret';

import styles from './chip.module.css';

export function Chip({
    children,
    identity,
    look = 'raised',
}: {
    children: string;
    identity?: Identity | undefined;
    look?: 'boxed' | 'raised';
}) {
    return (
        <span
            className={`${styles.chip} ${look === 'boxed' ? styles.boxed : ''}`}
            data-identity={identity}
        >
            {identity === undefined ? null : <span aria-hidden="true" className={styles.square} />}
            {shownOrWithheld(children)}
        </span>
    );
}
