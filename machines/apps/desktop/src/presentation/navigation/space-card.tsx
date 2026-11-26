import type { EngineReadOutput, MachinesMacView } from '@jterrazz/station-api';
import { useState } from 'react';

import type { Reading } from '../dependencies/dependencies';
import { Icon } from '../ui/icon';
import type { IconName } from '../ui/icon';
import { Popover } from '../ui/index';
import type { PopoverRow } from '../ui/index';
import type { Carried } from './carried';
import { engineMarked } from './engine-foot';
import { useEngineLine } from './engine-line';
import { machineRows, matches } from './machine-popover';
import type { Opened } from './machine-popover';
import { machineRoute } from './route';
import type { Route } from './route';
import { fleetRoute, SECTIONS } from './segments';
import type { Section } from './segments';
import { spaceOfRoute } from './spaces';
import { TITLES } from './titles';

const SECTION_ICONS: Readonly<Record<Section, IconName>> = {
    'catalog-apps': 'apps',
    'catalog-settings': 'settings',
    'catalog-skills': 'skills',
    engine: 'engine',
    fleet: 'server',
    gallery: 'box',
    home: 'home',
    inbox: 'inbox',
    'machine-files': 'file',
    passwords: 'key',
    settings: 'gear',
    storage: 'disc',
};

const FILTER = 'Go to a section or a machine';

// value: a row's id says its column, so a machine named as a section is still told apart.
const AT_SECTION = 'section:';
const AT_MACHINE = 'machine:';

export type SpaceCardProps = Opened & {
    label: string;
    route: Route;
    macs: readonly MachinesMacView[];
    carried: Carried;
    engine: Reading<EngineReadOutput>;
    lastMachine: () => Route | undefined;
    go: (to: Route) => void;
};

type Said = Pick<SpaceCardProps, 'carried' | 'lastMachine' | 'macs'> & {
    engine: PopoverRow['state'];
};

function hintOf(section: Section, carried: Carried): string | undefined {
    const counted: Partial<Record<Section, number | undefined>> = {
        'catalog-apps': carried.apps,
        'catalog-settings': carried.settings,
        'catalog-skills': carried.skills,
    };
    if (section === 'home') {
        return '⌘0';
    }
    const count = counted[section];
    return count === undefined ? undefined : String(count);
}

function currentOf(route: Route): Section {
    return route.page === 'machine' ? 'fleet' : route.page;
}

// value: the space's sections as the card lists them: their icon, Home's key, a shelf's count and the engine's mark
// value: when it is not fine; Fleet is greyed only when there is no machine to go to.
function sectionRows(route: Route, said: Said): readonly PopoverRow[] {
    const current = currentOf(route);
    const fleet = fleetRoute(said.lastMachine(), said.macs[0]?.Name);
    return SECTIONS[spaceOfRoute(route) ?? 'machine'].map((section) => ({
        current: section === current,
        disabled: section === 'fleet' && fleet === undefined,
        hint: hintOf(section, said.carried),
        id: `${AT_SECTION}${section}`,
        label: section === 'fleet' ? 'Fleet' : TITLES[section],
        lead: <Icon name={SECTION_ICONS[section]} />,
        state: section === 'engine' ? said.engine : undefined,
    }));
}

function routeOf(id: string, said: Said): Route | undefined {
    if (id.startsWith(AT_MACHINE)) {
        return machineRoute(id.slice(AT_MACHINE.length));
    }
    const section = Object.values(SECTIONS)
        .flat()
        .find((one) => `${AT_SECTION}${one}` === id);
    if (section === 'fleet') {
        return fleetRoute(said.lastMachine(), said.macs[0]?.Name);
    }
    return section === undefined ? undefined : { page: section };
}

function machinesSaid(count: number): string {
    return `${String(count)} ${count === 1 ? 'machine' : 'machines'}`;
}

// invariant: one filter for both columns: a column it empties says so in its place, both emptied say Nothing
// invariant: matches; a machine it keeps is listed flat, as a Mac it hides would leave its VMs under nothing.
function cardSections(route: Route, said: Said, query: string) {
    const current = route.page === 'machine' ? route.machine : undefined;
    const machines = machineRows({ current, macs: said.macs }).map((row) => ({
        ...row,
        id: `${AT_MACHINE}${row.id}`,
    }));
    const typed = query.trim() !== '';
    const sections = sectionRows(route, said).filter((row) => matches(row.label, query));
    const fleet = typed
        ? machines
              .filter((row) => matches(row.label, query))
              .map((row) => ({ ...row, indent: false }))
        : machines;
    return [
        { empty: 'No section matches', label: 'Sections', rows: sections },
        {
            empty: 'No machine matches',
            label: 'Fleet',
            meta: machinesSaid(machines.length),
            rows: fleet,
        },
    ];
}

// value: the card the path's first segment opens: the space's sections beside the whole fleet, under one filter,
// value: on the window's glass; the route's section and the open machine are its current rows.
export function SpaceCard(props: SpaceCardProps) {
    const { anchor, id, onClose, label, route, macs, carried, engine, lastMachine, go } = props;
    const [query, setQuery] = useState('');
    const line = useEngineLine(engine);
    const marked = engineMarked({ engine, quiet: false }, line) ? line.dot : undefined;
    const said: Said = { carried, engine: marked, lastMachine, macs };
    return (
        <Popover
            anchor={anchor}
            columns
            empty="Nothing matches"
            filter={{ label: FILTER, onChange: setQuery, placeholder: FILTER, value: query }}
            id={id}
            label={label}
            onChoose={(chosen) => {
                const to = routeOf(chosen, said);
                if (to !== undefined) {
                    go(to);
                }
            }}
            onClose={onClose}
            sections={cardSections(route, said, query)}
        />
    );
}
