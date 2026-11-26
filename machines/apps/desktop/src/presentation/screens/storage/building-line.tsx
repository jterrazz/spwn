import type { ImagesBuilding, ImagesReport } from '@jterrazz/station-api';

import { sentence } from '../../../domain/format';
import { Icon } from '../../ui/icon';
import { Row, StateMark } from '../../ui/index';
import { wordOf } from '../../words/engine-words';
import { engineShown, newestState } from '../../words/shown';

import styles from '../../features/disk/disk.module.css';

type BuildingLineProps = { building: ImagesBuilding; image: ImagesReport };

function Walk({ building }: { building: ImagesBuilding }) {
    const walk = building.Walk ?? [];
    return walk.length === 0 ? null : (
        <span className={styles.walk}>
            {walk.map((step) => (
                <span className={styles.step} key={step.Step}>
                    <StateMark said="name" {...engineShown('ImagesBuildState', step.State)} />
                    {step.Word}
                </span>
            ))}
        </span>
    );
}

export function BuildingLine({ building, image }: BuildingLineProps) {
    const newest = newestState(image.Newest);
    return (
        <Row
            below={<Walk building={building} />}
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
                        <StateMark {...newest} />
                    </span>
                )
            }
            trailSize="wide"
        />
    );
}
