import { useState } from 'react';

import { Popover } from '../ui/index';
import { matches } from './machine-popover';
import type { Opened } from './machine-popover';
import type { Route } from './route';

export type Way = { id: string; label: string; to: Route };

type WaysPopoverProps = Opened & {
    label: string;
    ways: readonly Way[];
    go: (to: Route) => void;
};

// value: the levels the fold hides, each a row that goes to it.
export function WaysPopover({ anchor, id, onClose, label, ways, go }: WaysPopoverProps) {
    const [query, setQuery] = useState('');
    const rows = ways
        .filter((way) => matches(way.label, query))
        .map((way) => ({ id: way.id, label: way.label }));
    return (
        <Popover
            anchor={anchor}
            empty="Nothing matches"
            filter={{
                label: `Filter ${label.toLowerCase()}`,
                onChange: setQuery,
                placeholder: `Filter ${label.toLowerCase()}`,
                value: query,
            }}
            id={id}
            label={label}
            onChoose={(chosen) => {
                const way = ways.find((one) => one.id === chosen);
                if (way !== undefined) {
                    go(way.to);
                }
            }}
            onClose={onClose}
            sections={[{ label, rows }]}
        />
    );
}
