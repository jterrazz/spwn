import { clock } from '../../domain/format';
import { SEPARATOR } from '../../domain/separator';
import { useStale } from '../dependencies/dependencies';
import { shownOrWithheld } from '../ui/secret';
import { StateMark } from '../ui/state-mark';
import { ownState } from '../words/shown';
import { asOfWord } from '../words/words';

import styles from './path-bar.module.css';

// value: one dimmed dot in the bar while a read drawn keeps an answer its refresh could not replace, named by
// value: what the engine said of each refusal, and the time of the oldest answer still shown beside it; the page
// value: itself stays drawn as it was answered.
export function StaleMark() {
    const refused = useStale();
    if (refused.length === 0) {
        return null;
    }
    const { tone, word } = ownState('stale');
    const said = [
        ...new Set(
            refused.map(({ error }) => shownOrWithheld(error.wire?.message ?? error.message)),
        ),
    ].filter((message) => message !== '');
    const named = said.length === 0 ? word : said.join(SEPARATOR);
    const oldest = new Date(Math.min(...refused.map(({ at }) => at.getTime())));
    return (
        <span className={styles.stale}>
            <span aria-label={named} role="status">
                <span aria-hidden="true">
                    <StateMark said="name" tone={tone} word={named} />
                </span>
            </span>
            <span className={styles.asOf}>{asOfWord(clock(oldest) ?? '')}</span>
        </span>
    );
}
