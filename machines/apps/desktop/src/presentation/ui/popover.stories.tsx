import { useState } from 'react';

import type { Tone } from '../../domain/tone';
import { capitalised } from './case';
import { Icon } from './icon';
import type { Identity } from './identity';
import { MachineIcon } from './machine-icon';
import type { MachineKind } from './machine-icon';
import { Popover, PopoverHost, usePopover } from './popover';
import type { PopoverProps, PopoverRow } from './popover';
import { nothing } from './story';
import type { Story } from './story';

import styles from './story.module.css';

export type PopoverSampleProps = {
    opener: string;
    at?: 'end' | 'start';
    popover: Omit<PopoverProps, 'anchor' | 'id' | 'onClose'>;
};

// value: a popover as a page opens it, under its opener, held in a frame of its own so the gallery draws each one
// value: where it stands rather than over the whole page.
export function PopoverSample({ opener, at = 'start', popover }: PopoverSampleProps) {
    const [host, setHost] = useState<HTMLDivElement | null>(null);
    const { anchor, id, open, opener: opens } = usePopover(true);
    return (
        <div className={`${styles.popover} ${at === 'end' ? styles.popoverEnd : ''}`} ref={setHost}>
            <button className={styles.opener} type="button" {...opens}>
                {opener}
            </button>
            {host && open ? (
                <PopoverHost value={host}>
                    <Popover {...popover} anchor={anchor} id={id} onClose={nothing} />
                </PopoverHost>
            ) : null}
        </div>
    );
}

type Machine = [name: string, kind: MachineKind, identity: Identity, state: string];

const TONE_OF: Readonly<Record<string, Tone>> = {
    asleep: 'plain',
    creating: 'new',
    down: 'bad',
    stopped: 'plain',
};

function rowOf([name, kind, identity, state]: Machine, indent: boolean): PopoverRow {
    return {
        id: name,
        indent,
        label: name,
        lead: <MachineIcon identity={kind === 'mac' ? undefined : identity} kind={kind} />,
        state: { tone: TONE_OF[state] ?? 'ok', word: capitalised(state) },
    };
}

const FLEET: readonly (readonly [Machine, readonly Machine[]])[] = [
    [
        ['macbook-pro', 'mac', 'system', 'up'],
        [
            ['agent-7f', 'workspace', 'blue', 'running'],
            ['api', 'workspace', 'purple', 'stopped'],
            ['linux-1', 'workspace', 'teal', 'running'],
            ['linux-2', 'workspace', 'indigo', 'creating'],
        ],
    ],
    [
        ['mac-mini', 'mac', 'system', 'up'],
        [
            ['infra', 'server', 'pink', 'down'],
            ['linux-3', 'workspace', 'sky', 'running'],
            ['review-bot', 'workspace', 'lilac', 'running'],
        ],
    ],
    [['macbook-air', 'mac', 'system', 'asleep'], []],
];

const MACHINES: readonly PopoverRow[] = FLEET.flatMap(([mac, vms]) => [
    rowOf(mac, false),
    ...vms.map((vm) => ({ ...rowOf(vm, true), current: vm[0] === 'linux-1' })),
]);

const FILTER = { label: 'Filter machines', onChange: nothing, placeholder: 'Filter machines' };

const CHOOSING = { empty: 'No machine matches', label: 'Machines', onChoose: nothing };

const MANY: readonly PopoverRow[] = Array.from({ length: 30 }, (_, at) =>
    rowOf(
        [`linux-${String(at + 1)}`, 'workspace', 'teal', at % 7 === 3 ? 'stopped' : 'running'],
        false,
    ),
);

function spaceOf(id: 'agent' | 'machine' | 'os', label: string, line: string): PopoverRow {
    const soon = id !== 'machine';
    return {
        current: !soon,
        description: soon ? `Soon · ${line}` : line,
        disabled: soon,
        id,
        label,
        lead: <Icon name={id} />,
        state: soon ? undefined : { tone: 'bad', word: 'Down' },
    };
}

export const POPOVER_STORIES: readonly Story<PopoverSampleProps>[] = [
    {
        name: 'the machines and their states',
        props: {
            opener: 'linux-1',
            popover: {
                ...CHOOSING,
                filter: { ...FILTER, value: '' },
                sections: [{ label: 'Machines', rows: MACHINES }],
            },
        },
    },
    {
        name: 'filtered',
        props: {
            opener: 'linux-1',
            popover: {
                ...CHOOSING,
                filter: { ...FILTER, value: 'lin' },
                sections: [
                    {
                        label: 'Machines',
                        rows: MACHINES.filter((row) => row.label.includes('lin')).map((row) => ({
                            ...row,
                            indent: false,
                        })),
                    },
                ],
            },
        },
    },
    {
        name: 'a filter that matches nothing',
        props: {
            opener: 'linux-1',
            popover: {
                ...CHOOSING,
                filter: { ...FILTER, value: 'gate9' },
                sections: [{ label: 'Machines', rows: [] }],
            },
        },
    },
    {
        name: 'a long list scrolls',
        props: {
            opener: 'macbook-pro',
            popover: {
                ...CHOOSING,
                filter: { ...FILTER, value: '' },
                sections: [{ label: 'Machines', rows: MANY }],
            },
        },
    },
    {
        name: 'the spaces, Agent and OS greyed',
        props: {
            opener: 'Machine',
            popover: {
                empty: 'No space',
                label: 'Spaces',
                onChoose: nothing,
                sections: [
                    {
                        label: 'Spaces',
                        rows: [
                            spaceOf('machine', 'Machine', 'The fleet, access, security, documents'),
                            spaceOf('agent', 'Agent', 'The workforce: personalities, roles, focus'),
                            spaceOf('os', 'OS', 'Flows, projects, vision'),
                        ],
                    },
                ],
            },
        },
    },
    {
        name: 'at the window edge, shifted inside it',
        props: {
            at: 'end',
            opener: 'neo waits for your yes',
            popover: {
                empty: 'Nothing waits',
                label: 'Needs',
                onChoose: nothing,
                sections: [
                    {
                        label: 'Needs you',
                        rows: [
                            {
                                id: 'neo',
                                label: 'neo waits for your yes',
                                state: { tone: 'warn', word: 'Waiting' },
                            },
                            {
                                id: 'port',
                                label: 'port 5000 is reachable outside the tailnet',
                                state: { tone: 'bad', word: 'Exposed' },
                            },
                        ],
                    },
                ],
            },
        },
    },
];
