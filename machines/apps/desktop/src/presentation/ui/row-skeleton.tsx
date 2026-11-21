import { Bone } from './bone';
import type { RowProps } from './row';

import styles from './row.module.css';

export type RowSkeletonProps = {
    lead?: 'icon' | 'machine' | 'value';
    sentence?: boolean;
    trail?: NonNullable<RowProps['trailSize']>;
    size?: 'regular' | 'slim';
    template?: RowProps['template'];
};

const LEADS = { icon: styles.leadIcon, machine: styles.leadMachine, value: styles.leadValue };
const TRAILS = { fit: styles.hasTrail, pill: styles.trailPill, wide: styles.trailWide };

// value: a row before its answer, drawn in the row's own box and columns: a title line and, where its rows carry
// value: one, a sentence line under it, so it stands as tall as the rows that replace it, its columns where theirs are.
export function RowSkeleton({
    lead,
    sentence = true,
    trail,
    size = 'regular',
    template = 'flow',
}: RowSkeletonProps) {
    const classes = [
        styles.row,
        styles.grid,
        size === 'slim' ? styles.slim : undefined,
        template === 'flow' ? undefined : styles[template],
        lead === undefined ? undefined : LEADS[lead],
        trail === undefined ? undefined : TRAILS[trail],
    ].filter((name) => name !== undefined);
    return (
        <div aria-hidden="true" className={classes.join(' ')}>
            {lead === undefined ? null : (
                <div className={styles.lead}>
                    <Bone width="short" />
                </div>
            )}
            <div className={styles.text}>
                <span className={styles.title}>
                    <Bone width="long" />
                </span>
                {sentence ? (
                    <span className={styles.sentence}>
                        <Bone width="wide" />
                    </span>
                ) : null}
            </div>
            {trail === undefined ? null : (
                <div className={styles.end}>
                    <Bone width="short" />
                </div>
            )}
        </div>
    );
}
