import { asSentence } from './case';
import { shownOrWithheld } from './secret';

import styles from './progress-line.module.css';

export type ProgressLineProps = {
    task: string;
    detail?: string | undefined;
    failed?: boolean;
};

export function ProgressLine({ task, detail, failed = false }: ProgressLineProps) {
    return (
        <span className={`${styles.line} ${failed ? styles.failed : ''}`} role="status">
            {failed ? null : <i aria-hidden="true" className={styles.spinner} />}
            <span className={styles.task}>{asSentence(task)}</span>
            {(detail ?? '') === '' ? null : (
                <span className={styles.detail}>{shownOrWithheld(detail ?? '')}</span>
            )}
        </span>
    );
}
