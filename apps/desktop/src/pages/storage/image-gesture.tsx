import type { CallName, ImagesReport, MachinesMacView } from '@jterrazz/station-api';

import { targetOf } from '../../domain/machines/machines';
import { SEPARATOR } from '../../domain/separator';
import { useGesture } from '../../presentation/dependencies/use-gesture';
import { Icon } from '../../presentation/ui/icon';
import { StatePill } from '../../presentation/ui/index';
import { isDeletable } from '../../presentation/words/provisional';
import { newestState } from '../../presentation/words/shown';
import { GestureButton, GestureRow } from '../kit/gestures';

import styles from '../disk/disk.module.css';

const DELETE_REREADS: readonly CallName[] = ['images.list', 'disk.read'];

type ImageLineProps = {
    image: ImagesReport;
    mac: MachinesMacView;
    sentence: string;
};

export function ImageLine({ image, mac, sentence }: ImageLineProps) {
    const { name, version } = image.Record;
    const shown = `${name} ${version}`;
    const newest = newestState(image.Newest);
    const remove = useGesture('images.delete', {
        input: { Image: name, Version: version },
        rereads: DELETE_REREADS,
        subject: shown,
        target: targetOf({ mac }),
    });
    return (
        <GestureRow
            gesture={remove}
            lead={
                <span className={styles.disc}>
                    <Icon name="disc" />
                </span>
            }
            sentence={sentence}
            tag={version}
            title={name}
            trail={
                <span className={styles.trail}>
                    {newest === null ? null : <StatePill {...newest} />}
                    {isDeletable(image) ? (
                        <GestureButton
                            destroys
                            gesture={remove}
                            label={
                                image.Record.size_gb > 0
                                    ? `Delete${SEPARATOR}frees ${String(image.Record.size_gb)} GB`
                                    : 'Delete'
                            }
                            name={`Delete ${shown}`}
                            title={`Delete ${shown}?`}
                            verb="Delete"
                        />
                    ) : null}
                </span>
            }
            trailSize="wide"
        />
    );
}
