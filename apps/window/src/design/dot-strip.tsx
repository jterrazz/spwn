import type { Tone } from '../tone';
import type { Identity } from './identity';

import styles from './dot-strip.module.css';

const NONE_HOLLOW: ReadonlyMap<string, Tone> = new Map();

type DotStripProps = {
    names: readonly string[];
    identity: Identity;
    hollow?: ReadonlyMap<string, Tone>;
};

export function DotStrip({ names, identity, hollow = NONE_HOLLOW }: DotStripProps) {
    return (
        <span aria-hidden="true" className={styles.strip} data-identity={identity}>
            {names.map((name) => {
                const tone = hollow.get(name);
                return (
                    <i
                        className={tone === undefined ? styles.dot : `${styles.dot} ${styles.hollow}`}
                        data-tone={tone}
                        key={name}
                        title={name}
                    />
                );
            })}
        </span>
    );
}
