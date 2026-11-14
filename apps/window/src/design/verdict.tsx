import type { ReactNode } from 'react';

import type { Tone } from '../tone';
import { StateDot } from './state-dot';

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
            <p className={styles.line}>
                <StateDot tone={tone} word={word} />
                <span>{line}</span>
            </p>
            {action === undefined ? null : <div className={styles.action}>{action}</div>}
        </div>
    );
}
