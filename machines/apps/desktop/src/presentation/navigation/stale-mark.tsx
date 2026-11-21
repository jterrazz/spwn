import { SEPARATOR } from '../../domain/separator';
import { useStale } from '../dependencies/dependencies';
import { shownOrWithheld } from '../ui/secret';
import { StateDot } from '../ui/state-dot';
import { ownState } from '../words/shown';

import styles from './toolbar.module.css';

// value: one dimmed dot in the toolbar while a read drawn keeps an answer its refresh could not replace, named by
// value: what the engine said of each refusal; the page itself stays drawn as it was answered.
export function StaleMark() {
    const refused = useStale();
    if (refused.length === 0) {
        return null;
    }
    const { tone, word } = ownState('stale');
    const said = [
        ...new Set(refused.map((error) => shownOrWithheld(error.wire?.message ?? error.message))),
    ].filter((message) => message !== '');
    const named = said.length === 0 ? word : said.join(SEPARATOR);
    return (
        <span aria-label={named} className={styles.stale} role="status">
            <span aria-hidden="true">
                <StateDot tone={tone} word={named} />
            </span>
        </span>
    );
}
