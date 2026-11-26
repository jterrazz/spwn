import { useState } from 'react';

import { SEPARATOR } from '../../domain/separator';
import { Icon } from '../ui/icon';
import { Popover } from '../ui/index';
import type { PopoverRow } from '../ui/index';
import type { Shown } from '../words/shown';
import { matches } from './machine-popover';
import type { Opened } from './machine-popover';
import { opensNow, SOON_WORD, SPACE_WORDS, SPACES } from './spaces';
import type { Space } from './spaces';

// value: a space's row: its mark, its name and its line; Agent and OS greyed, saying Soon, until they run.
export function spaceRows(standing: Shown | undefined): readonly PopoverRow[] {
    return SPACES.map((space) => {
        const words = SPACE_WORDS[space];
        const soon = !opensNow(space);
        return {
            current: space === 'machine',
            description: soon ? `${SOON_WORD}${SEPARATOR}${words.line}` : words.line,
            disabled: soon,
            id: space,
            label: words.name,
            lead: <Icon name={space} />,
            state: soon ? undefined : standing,
        };
    });
}

type SpacePopoverProps = Opened & {
    standing: Shown | undefined;
    onSpace: (space: Space) => void;
};

export function SpacePopover({ anchor, id, onClose, standing, onSpace }: SpacePopoverProps) {
    const [query, setQuery] = useState('');
    const rows = spaceRows(standing).filter((row) => matches(row.label, query));
    return (
        <Popover
            anchor={anchor}
            empty="No space matches"
            filter={{
                label: 'Filter spaces',
                onChange: setQuery,
                placeholder: 'Filter spaces',
                value: query,
            }}
            id={id}
            label="Spaces"
            onChoose={(chosen) => {
                const space = SPACES.find((one) => one === chosen);
                if (space !== undefined) {
                    onSpace(space);
                }
            }}
            onClose={onClose}
            sections={[{ label: 'Spaces', rows }]}
        />
    );
}
