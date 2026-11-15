import type { MachinefileDiffering } from '@jterrazz/station-api';

import { driftAsked } from '../../domain/machines/machines';
import type { Found } from '../../domain/machines/machines';
import { useRead } from '../../presentation/dependencies/dependencies';
import { AppIcon, Panel, StatePill, Value } from '../../presentation/ui/index';
import { accountSideOf } from '../../presentation/words/provisional';
import { differingSide } from '../../presentation/words/shown';
import type { DifferingSide, SharedChapter } from '../../presentation/words/shown';

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

function Side({
    chapter,
    differing,
    name,
    side,
}: {
    chapter: SharedChapter;
    differing: MachinefileDiffering;
    name: string;
    side: DifferingSide;
}) {
    const shown = differingSide(chapter, side, differing);
    const absent = chapter === 'accounts' && accountSideOf(side, differing) === 'not-in-the-file';
    if (absent) {
        return <span className={styles.hint}>{shown.word}</span>;
    }
    return (
        <div className={styles.item}>
            <Lead chapter={chapter} differing={differing} name={name} />
            <StatePill {...shown} dotted={false} />
        </div>
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
                        <Side
                            chapter={chapter}
                            differing={one}
                            key={side}
                            name={nameOf(one)}
                            side={side}
                        />
                    ))}
                </div>
            ))}
        </Panel>
    );
}
