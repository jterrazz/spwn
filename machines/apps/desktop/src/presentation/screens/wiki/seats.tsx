import type { WikiSeatHead } from '@jterrazz/station-api';

import { WIKI_WORDS } from '../../words/wiki';
import type { Grids } from './dot-icon';
import { DotIcon } from './dot-icon';

import styles from './wiki.module.css';

const ALL_GRID: Grids = { all: ['#.#.#', '.....', '#.#.#', '.....', '#.#.#'] };

type Seat = { id: string; name: string; icon: string; chosen: boolean; to: string | undefined };

// invariant: the seats bar is one row, All first; under 560 px the chosen seat keeps its name and the others show
// invariant: their icon alone, named by their aria-label.
export function SeatsBar({
    seats,
    grids,
    home,
    open,
}: {
    seats: readonly WikiSeatHead[];
    grids: Grids;
    home: boolean;
    open: (page: string | undefined) => void;
}) {
    const all: Seat = { chosen: home, icon: 'all', id: '', name: WIKI_WORDS.all, to: undefined };
    const listed: Seat[] = seats.map((seat) => ({
        chosen: !home && seat.Active,
        icon: seat.Icon,
        id: seat.ID,
        name: seat.Name,
        to: seat.Home,
    }));
    return (
        <nav aria-label={WIKI_WORDS.seats} className={styles.seats}>
            {[all, ...listed].map((seat) => (
                <button
                    aria-label={seat.name}
                    aria-pressed={seat.chosen}
                    className={`${styles.seat} ${seat.chosen ? styles.seatChosen : ''}`}
                    key={seat.id}
                    onClick={() => {
                        open(seat.to);
                    }}
                    type="button"
                >
                    <DotIcon grids={{ ...ALL_GRID, ...grids }} name={seat.icon} size="small" />
                    <span className={styles.seatName}>{seat.name}</span>
                </button>
            ))}
        </nav>
    );
}
