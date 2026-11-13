import { shownOrWithheld } from './secret';

import styles from './tag.module.css';

const KEPT_END = 18;

export function Value({ children }: { children: string }) {
    const shown = shownOrWithheld(children);
    const cut = Math.max(0, shown.length - Math.min(KEPT_END, Math.floor(shown.length / 2)));
    return (
        <span className={styles.value} title={shown}>
            <span className={styles.whole}>{shown}</span>
            <span aria-hidden className={styles.start}>
                {shown.slice(0, cut)}
            </span>
            <span aria-hidden className={styles.finish}>
                {shown.slice(cut)}
            </span>
        </span>
    );
}
