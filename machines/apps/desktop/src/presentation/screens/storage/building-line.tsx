import type { ImagesBuilding, ImagesReport } from '@jterrazz/station-api';

import { sentence } from '../../../domain/format';
import { Icon } from '../../ui/icon';
import { Row, StatePill } from '../../ui/index';
import { wordOf } from '../../words/engine-words';
import { newestState } from '../../words/shown';

import styles from '../../features/disk/disk.module.css';

type BuildingLineProps = { building: ImagesBuilding; image: ImagesReport };

export function BuildingLine({ building, image }: BuildingLineProps) {
    const newest = newestState(image.Newest);
    return (
        <Row
            lead={
                <span className={styles.disc}>
                    <Icon name="disc" />
                </span>
            }
            sentence={sentence([wordOf('ImagesBuildStep', building.Step).Word, building.Sentence])}
            tag={image.Record.version}
            title={image.Record.name}
            trail={
                newest === null ? undefined : (
                    <span className={styles.trail}>
                        <StatePill {...newest} />
                    </span>
                )
            }
            trailSize="wide"
        />
    );
}
