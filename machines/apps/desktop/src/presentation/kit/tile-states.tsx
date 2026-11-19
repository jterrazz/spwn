import { MissingMark, Tile } from '../ui/index';
import { ownState } from '../words/shown';
import type { OwnState } from '../words/words';

export function split(text: string): { value: string; unit: string } {
    const space = text.indexOf(' ');
    return space === -1
        ? { unit: '', value: text }
        : { unit: text.slice(space + 1), value: text.slice(0, space) };
}

export function Unknown({ label, state }: { label: string; state: OwnState }) {
    return <Tile foot="" label={label} value={<MissingMark>{ownState(state).word}</MissingMark>} />;
}

export function NotGivenTile({ label }: { label: string }) {
    return <Tile foot="" label={label} value={<MissingMark />} />;
}
