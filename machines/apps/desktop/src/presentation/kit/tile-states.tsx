import { MissingMark, Tile } from '../ui/index';
import { ownState } from '../words/shown';
import type { OwnState } from '../words/words';
import { SkeletonTile } from './skeleton';

export function split(text: string): { value: string; unit: string } {
    const space = text.indexOf(' ');
    return space === -1
        ? { unit: '', value: text }
        : { unit: text.slice(space + 1), value: text.slice(0, space) };
}

// value: a tile not answered yet keeps its label and draws its figure and foot as bones, never the word "reading".
export function Unknown({ label, state }: { label: string; state: OwnState }) {
    if (state === 'reading') {
        return <SkeletonTile label={label} />;
    }
    return <Tile foot="" label={label} value={<MissingMark>{ownState(state).word}</MissingMark>} />;
}

export function NotGivenTile({ label }: { label: string }) {
    return <Tile foot="" label={label} value={<MissingMark />} />;
}
