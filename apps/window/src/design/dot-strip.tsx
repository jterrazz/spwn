import type { Identity } from './identity';

import styles from './dot-strip.module.css';

export function DotStrip({ names, identity }: { names: readonly string[]; identity: Identity }) {
    return (
        <span aria-hidden="true" className={styles.strip} data-identity={identity}>
            {names.map((name) => (
                <i className={styles.dot} key={name} title={name} />
            ))}
        </span>
    );
}
