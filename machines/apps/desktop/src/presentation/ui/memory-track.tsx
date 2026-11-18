import type { Identity } from './identity';
import type { Measured } from './measure';

import styles from './memory-track.module.css';

// value: a segment of a Mac's memory, its offset and width as shares of the Mac's total, and its used part as a
// value: share of the segment itself; a segment whose use is unknown is drawn reserved and never filled.
export type TrackSegment = {
    key: string;
    at: number;
    width: number;
    used?: number | undefined;
    identity?: Identity | undefined;
};

export type MemoryTrackProps = {
    label: string;
    segments: readonly TrackSegment[];
};

function share(value: number): number {
    return Number.isFinite(value) ? Math.max(0, Math.min(1, value)) : 0;
}

function placeOf({ at, width, used }: TrackSegment): Measured {
    const start = share(at);
    const placed: Measured = {
        '--at': Number(start.toFixed(4)),
        '--w': Number(Math.min(share(width), 1 - start).toFixed(4)),
    };
    return used === undefined ? placed : { ...placed, '--used': Number(share(used).toFixed(4)) };
}

export function MemoryTrack({ label, segments }: MemoryTrackProps) {
    return (
        <div aria-label={label} className={styles.track} role="img">
            {segments.map((segment) => (
                <i
                    className={styles.segment}
                    data-identity={segment.identity}
                    key={segment.key}
                    style={placeOf(segment)}
                />
            ))}
        </div>
    );
}
