import type { DiskOutcome } from '@jterrazz/station-api';

import { StatePill, Value } from '../../design/index';
import { DISK_RESULT_TONES } from '../../engine/tones';
import { DISK_RESULT_WORDS } from '../../engine/words';
import { bytes, sentence } from '../../model/format';
import type { Held } from '../kit/gestures';

export function OutcomePill({ outcome }: { outcome: DiskOutcome }) {
    return (
        <StatePill
            tone={DISK_RESULT_TONES[outcome.Result]}
            word={DISK_RESULT_WORDS[outcome.Result]}
        />
    );
}

export function FreedFigure({ outcome }: { outcome: DiskOutcome }) {
    return <Value>{`${bytes(outcome.Freed)} freed`}</Value>;
}

export function heldOf(outcome: DiskOutcome): Held {
    return {
        sentence: sentence([outcome.Detail, outcome.Reason, outcome.Hint]),
        shown: (
            <>
                <FreedFigure outcome={outcome} />
                <OutcomePill outcome={outcome} />
            </>
        ),
        title: outcome.Name,
    };
}
