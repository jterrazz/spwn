import type { CallName, ImagesReport, MachinesMacView } from '@jterrazz/station-api';

import { AppIcon, StatePill } from '../../design/index';
import { newestState } from '../../engine/shown';
import { useGesture } from '../../engine/use-gesture';
import { targetOf } from '../../model/machines';
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
            lead={<AppIcon name={name} />}
            sentence={sentence}
            tag={version}
            title={name}
            trail={
                <span className={styles.trail}>
                    {newest === null ? null : <StatePill {...newest} />}
                    <GestureButton
                        gesture={remove}
                        label="Delete"
                        name={`Delete ${shown}`}
                        title={`Delete ${shown}?`}
                        variant="danger"
                    />
                </span>
            }
            trailSize="wide"
        />
    );
}
