import type { MachinefileDiffering } from '@jterrazz/station-api';

import { AppIcon, Panel, StatePill, Value } from '../../design/index';
import { useRead } from '../../engine/react';
import { differingSide } from '../../engine/shown';
import type { DifferingSide, SharedChapter } from '../../engine/shown';
import { driftAsked } from '../../model/machines';
import type { Found } from '../../model/machines';

import styles from './chapter-differs.module.css';

const SIDES: readonly DifferingSide[] = ['asked', 'seen'];

function Lead({
    chapter,
    differing,
    name,
}: {
    chapter: SharedChapter;
    differing: MachinefileDiffering;
    name: string;
}) {
    return (
        <>
            {chapter === 'ports' ? (
                <Value look="raised">{`:${differing.Port}`}</Value>
            ) : (
                <AppIcon name={differing.Account} />
            )}
            {name === '' ? null : <span className={styles.name}>{name}</span>}
        </>
    );
}

type ChapterDiffersProps = {
    found: Found;
    chapter: SharedChapter;
    nameOf: (differing: MachinefileDiffering) => string;
};

export function ChapterDiffers({ found, chapter, nameOf }: ChapterDiffersProps) {
    const drift = useRead('machinefile.drift', driftAsked(found));
    if (drift.kind !== 'read') {
        return null;
    }
    const differing =
        (drift.out.Chapters ?? []).find((one) => one.Chapter === chapter)?.Differing ?? [];
    if (differing.length === 0) {
        return null;
    }
    return (
        <Panel note={String(differing.length)} title="Differs from its file">
            <div className={`${styles.compare} ${styles.head}`}>
                <span>The file says</span>
                <span>This machine has</span>
            </div>
            {differing.map((one) => (
                <div className={styles.compare} key={`${one.Port}/${one.Account}`}>
                    {SIDES.map((side) => (
                        <div className={styles.item} key={side}>
                            <Lead chapter={chapter} differing={one} name={nameOf(one)} />
                            <StatePill {...differingSide(chapter, side, one)} dotted={false} />
                        </div>
                    ))}
                </div>
            ))}
        </Panel>
    );
}
