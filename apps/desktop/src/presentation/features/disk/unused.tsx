import type { DiskCanFree } from '@jterrazz/station-api';

import { bytes } from '../../../domain/format';
import { Icon } from '../../ui/icon';
import { Panel, Row } from '../../ui/index';
import { wordOfSent } from '../../words/engine-words';

import styles from './disk.module.css';

export function UnusedFoot({ canFree }: { canFree: DiskCanFree }) {
    return (
        <div className={styles.foot}>
            <p>{canFree.Sentence}</p>
        </div>
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
                    lead={
                        <span className={styles.disc}>
                            <Icon name="disc" />
                        </span>
                    }
                    sentence={wordOfSent('DiskHolder', one.Holder)}
                    title={one.Name}
                    value={bytes(one.Bytes)}
                />
            ))}
        </Panel>
    );
}
