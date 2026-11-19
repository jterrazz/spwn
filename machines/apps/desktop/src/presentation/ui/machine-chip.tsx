import type { Identity } from './identity';
import { MachineIcon } from './machine-icon';
import type { MachineKind } from './machine-icon';

import styles from './machine-chip.module.css';

export type MachineChipProps = {
    name: string;
    kind: MachineKind;
    identity?: Identity | undefined;
    onOpen: () => void;
};

export function MachineChip({ name, kind, identity, onOpen }: MachineChipProps) {
    return (
        <button className={styles.chip} onClick={onOpen} type="button">
            <MachineIcon identity={identity} kind={kind} />
            {name}
        </button>
    );
}
