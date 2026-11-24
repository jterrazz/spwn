import type { CallName, DiskCanFree, DiskKept, Target } from '@jterrazz/station-api';

import { bytes } from '../../../domain/format';
import { useGesture } from '../../dependencies/use-gesture';
import type { ButtonVariant } from '../../ui/button';
import { Icon } from '../../ui/icon';
import { Panel, Row } from '../../ui/index';
import { wordOfSent } from '../../words/engine-words';
import { DISK_GESTURE_LABELS, DISK_GESTURE_VERBS } from '../../words/words';
import { FreeAllButton } from './disk-gestures';
import type { DiskGesture } from './disk-gestures';

import styles from './disk.module.css';

const FREE_UNUSED_REREADS: readonly CallName[] = ['disk.read', 'images.list'];

export function useFreeUnused(canFree: DiskCanFree | undefined, target: Target | undefined) {
    return useGesture('disk.free-unused', {
        input: { Names: (canFree?.Frees ?? []).map((one) => one.Name) },
        rereads: FREE_UNUSED_REREADS,
        subject: undefined,
        target,
    });
}

type FootProps = {
    canFree: DiskCanFree;
    gesture: DiskGesture;
    name: string;
    variant: ButtonVariant;
};

export function UnusedFoot({ canFree, gesture, name, variant }: FootProps) {
    const label = `${DISK_GESTURE_LABELS['disk.free-unused']} ${bytes(canFree.Bytes)}`;
    return (
        <div className={styles.foot}>
            <p>{canFree.Sentence}</p>
            {canFree.Call === undefined ? null : (
                <FreeAllButton
                    gesture={gesture}
                    label={label}
                    name={name}
                    variant={variant}
                    verb={DISK_GESTURE_VERBS['disk.free-unused']}
                />
            )}
        </div>
    );
}

function Disc() {
    return (
        <span className={styles.disc}>
            <Icon name="disc" />
        </span>
    );
}

export function UnusedPanel({ canFree }: { canFree: DiskCanFree }) {
    const frees = canFree.Frees ?? [];
    if (frees.length === 0) {
        return null;
    }
    return (
        <Panel title="Can be freed">
            {frees.map((one) => (
                <Row
                    key={`${one.Holder}/${one.Name}`}
                    lead={<Disc />}
                    sentence={wordOfSent('DiskFreeableKind', one.Kind)}
                    title={one.Name}
                    value={bytes(one.Bytes)}
                />
            ))}
        </Panel>
    );
}

export function KeptPanel({ kept }: { kept: readonly DiskKept[] | undefined }) {
    if (kept === undefined || kept.length === 0) {
        return null;
    }
    return (
        <Panel title="Kept">
            {kept.map((one) => (
                <Row key={one.Name} lead={<Disc />} sentence={one.Sentence} title={one.Name} />
            ))}
        </Panel>
    );
}
