import { Icon } from './icon';
import type { Identity } from './identity';

import styles from './machine-icon.module.css';

export type MachineKind = 'mac' | 'server' | 'workspace';

export function MachineIcon({
    kind,
    identity,
    size = 'small',
}: {
    kind: MachineKind;
    identity?: Identity | undefined;
    size?: 'large' | 'small';
}) {
    return (
        <span
            aria-hidden="true"
            className={`${styles.icon} ${size === 'large' ? styles.large : ''}`}
            data-identity={identity}
        >
            <Icon name={kind} />
        </span>
    );
}
