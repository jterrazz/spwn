import { useState } from 'react';

import { Icon } from '../ui/icon';
import type { IconName } from '../ui/icon';
import { Popover } from '../ui/index';
import { matches } from './machine-popover';
import type { Opened } from './machine-popover';
import type { PlainPage, Route } from './route';
import { PLACES } from './segments';
import { TITLES } from './titles';

const PLACE_ICONS: Readonly<Partial<Record<PlainPage, IconName>>> = {
    'catalog-apps': 'apps',
    'catalog-settings': 'settings',
    'catalog-skills': 'skills',
    engine: 'engine',
    'machine-files': 'file',
    passwords: 'key',
    storage: 'disc',
};

// value: one row a popover lists that goes to a route: a place of the space, or a segment the fold hides.
export type Way = { id: string; label: string; to: Route; icon?: IconName | undefined };

export function placeWays(): readonly Way[] {
    return PLACES.map((page) => ({
        icon: PLACE_ICONS[page],
        id: page,
        label: TITLES[page],
        to: { page },
    }));
}

type WaysPopoverProps = Opened & {
    label: string;
    ways: readonly Way[];
    current: string | undefined;
    go: (to: Route) => void;
};

export function WaysPopover({ anchor, id, onClose, label, ways, current, go }: WaysPopoverProps) {
    const [query, setQuery] = useState('');
    const rows = ways
        .filter((way) => matches(way.label, query))
        .map((way) => ({
            current: way.id === current,
            id: way.id,
            label: way.label,
            lead: way.icon === undefined ? undefined : <Icon name={way.icon} />,
        }));
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
