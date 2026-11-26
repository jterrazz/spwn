import type { ReactNode } from 'react';

import type { Tone } from '../../domain/tone';
import { Bone } from './bone';
import { StateMark } from './state-mark';

import styles from './verdict.module.css';

export type VerdictProps = {
    title: ReactNode;
    line: ReactNode;
    tone: Tone;
    word: string;
    size: 'page' | 'tab';
    action?: ReactNode;
};

export function Verdict({ title, line, tone, word, size, action }: VerdictProps) {
    return (
        <div className={`${styles.verdict} ${styles[size]}`}>
            <h1 className={styles.title}>{title}</h1>
            {line === '' || line === undefined || line === null ? null : (
                <p className={styles.line}>
                    <StateMark said="name" tone={tone} word={word} />
                    <span>{line}</span>
                </p>
            )}
            {action === undefined ? null : <div className={styles.action}>{action}</div>}
        </div>
    );
}

export type VerdictSkeletonProps = { size: 'page' | 'tab'; line?: boolean };

// value: a verdict of one title line and, where its answer carries one, a line under it, before its answer: its
// value: height is an answered one's.
export function VerdictSkeleton({ size, line = true }: VerdictSkeletonProps) {
    return (
        <div aria-hidden="true" className={`${styles.verdict} ${styles[size]}`}>
            <p className={styles.title}>
                <Bone width="wide" />
            </p>
            {line ? (
                <p className={styles.line}>
                    <span className={styles.dot} />
                    <Bone width="long" />
                </p>
            ) : null}
        </div>
    );
}
