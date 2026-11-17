import { Button } from './button';
import type { FleetBoardProps, FleetMac } from './fleet-board';
import {
    AGENT,
    AIR,
    ALL_RUNNING,
    API,
    denseMac,
    DOWN_INFRA,
    FLEET,
    LINUX_1,
    LINUX_2,
    LINUX_3,
    macOf,
    MINI,
    MINI_MACHINES,
    PRO,
    REVIEW_BOT,
    STUDIO_MAC,
} from './fleet-sample';
import { nothing } from './story';
import type { Story } from './story';

const ADD_A_MAC = <Button variant="link">Add a Mac</Button>;

function board(
    macs: readonly FleetMac[],
    summary: string,
    more: Partial<FleetBoardProps> = {},
): FleetBoardProps {
    return { macs, onOpen: nothing, onOpenMac: nothing, summary, thisMac: 'this Mac', ...more };
}

export const FLEET_BOARD_STORIES: readonly Story<FleetBoardProps>[] = [
    {
        name: 'one Mac, no machine yet',
        props: board([macOf({ ...AIR, thisMac: true })], '0 machines · 1 Mac', { foot: ADD_A_MAC }),
    },
    {
        name: 'all running',
        props: board(
            [
                macOf({ ...MINI, thisMac: true }, MINI_MACHINES),
                macOf({ ...AIR, kind: '1 machine', state: ALL_RUNNING }, [AGENT]),
            ],
            '4 machines · 2 Macs',
        ),
    },
    {
        name: 'one needs you',
        props: board(
            [macOf({ ...PRO, kind: '2 machines' }, [AGENT, LINUX_1])],
            '2 machines · 1 Mac',
        ),
    },
    {
        name: 'creating',
        props: board(
            [
                macOf(
                    { ...PRO, kind: '2 machines', state: { tone: 'new', word: '1 being created' } },
                    [AGENT, LINUX_2],
                ),
            ],
            '2 machines · 1 Mac',
        ),
    },
    {
        name: 'broken',
        props: board(
            [
                macOf({ ...MINI, state: { tone: 'bad', word: '1 is down' }, thisMac: true }, [
                    DOWN_INFRA,
                    LINUX_3,
                    REVIEW_BOT,
                ]),
            ],
            '3 machines · 1 Mac',
        ),
    },
    {
        name: 'stopped: no segment',
        props: board(
            [macOf({ ...PRO, kind: '2 machines', state: ALL_RUNNING }, [AGENT, API])],
            '2 machines · 1 Mac',
        ),
    },
    {
        name: 'a Mac not seen: no room',
        props: board(
            [macOf({ ...MINI, thisMac: true }, MINI_MACHINES), STUDIO_MAC],
            '5 machines · 2 Macs',
        ),
    },
    {
        name: 'engine stopped',
        props: board(
            [
                macOf({
                    ...PRO,
                    cpu: '',
                    kind: 'Mac',
                    measured: false,
                    state: { tone: 'bad', word: 'engine stopped' },
                }),
                macOf(MINI, MINI_MACHINES),
            ],
            '3 machines · 2 Macs',
        ),
    },
    {
        name: 'dense fleet: 4 Macs, 8 machines each, a long name',
        props: board(
            ['studio-1', 'studio-2', 'studio-3', 'studio-4'].map(denseMac),
            '32 machines · 4 Macs',
            { foot: ADD_A_MAC },
        ),
    },
    {
        name: 'a Mac chosen',
        props: board(FLEET, '7 machines · 3 Macs', { chosenAtFirst: 'mac-mini', foot: ADD_A_MAC }),
    },
    {
        name: 'a row opened',
        props: board(FLEET, '7 machines · 3 Macs', { foot: ADD_A_MAC, openedAtFirst: 'linux-1' }),
    },
];
