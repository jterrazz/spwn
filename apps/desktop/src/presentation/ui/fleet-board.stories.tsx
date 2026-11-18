import { Button } from './button';
import type { FleetBoardProps, FleetMac, FleetState } from './fleet-board';
import {
    AGENT,
    AIR,
    ALL_RUNNING,
    API,
    ASLEEP,
    denseMac,
    DOWN_INFRA,
    ENGINE_NOT_INSTALLED,
    ENGINE_STOPPED,
    FLEET,
    LINUX_2,
    LINUX_3,
    macOf,
    MINI,
    MINI_MACHINES,
    NO_MACHINE_YET,
    ONE_DOWN,
    ONE_MAC_NOT_SEEN,
    PRO,
    RESTARTED,
    REVIEW_BOT,
    SLEEPER,
    STUDIO_MAC,
} from './fleet-sample';
import { nothing } from './story';
import type { Story } from './story';

const ADD_A_MAC = <Button variant="link">Add a Mac</Button>;

function board(
    macs: readonly FleetMac[],
    standing: FleetState,
    more: Partial<FleetBoardProps> = {},
): FleetBoardProps {
    return { macs, onOpen: nothing, onOpenMac: nothing, standing, thisMac: 'this Mac', ...more };
}

export const FLEET_BOARD_STORIES: readonly Story<FleetBoardProps>[] = [
    {
        name: 'one Mac, no machine yet',
        props: board([macOf({ ...AIR, thisMac: true })], NO_MACHINE_YET, { foot: ADD_A_MAC }),
    },
    {
        name: 'all running, a server healthy',
        props: board(
            [
                macOf({ ...MINI, thisMac: true }, MINI_MACHINES),
                macOf({ ...AIR, kind: '1 machine', state: ALL_RUNNING }, [AGENT]),
            ],
            ALL_RUNNING,
        ),
    },
    {
        name: 'a machine down',
        props: board(
            [macOf({ ...MINI, state: ONE_DOWN, thisMac: true }, [DOWN_INFRA, LINUX_3, REVIEW_BOT])],
            ONE_DOWN,
        ),
    },
    {
        name: 'creating and restarting',
        props: board(
            [macOf({ ...PRO, kind: '3 machines' }, [AGENT, LINUX_2, RESTARTED])],
            ALL_RUNNING,
        ),
    },
    {
        name: 'stopped and suspended: no segment',
        props: board([macOf({ ...PRO, kind: '3 machines' }, [AGENT, API, SLEEPER])], ALL_RUNNING),
    },
    {
        name: 'a Mac not seen: no room',
        props: board(
            [macOf({ ...MINI, thisMac: true }, MINI_MACHINES), STUDIO_MAC],
            ONE_MAC_NOT_SEEN,
        ),
    },
    {
        name: 'a Mac asleep',
        props: board(
            [macOf({ ...MINI, thisMac: true }, MINI_MACHINES), macOf({ ...AIR, state: ASLEEP })],
            ALL_RUNNING,
        ),
    },
    {
        name: 'engine stopped',
        props: board(
            [
                macOf({ ...PRO, cpu: '', kind: undefined, measured: false, state: ENGINE_STOPPED }),
                macOf(MINI, MINI_MACHINES),
            ],
            ONE_DOWN,
        ),
    },
    {
        name: 'engine not installed',
        props: board(
            [
                macOf({
                    ...PRO,
                    cpu: '',
                    kind: undefined,
                    measured: false,
                    state: ENGINE_NOT_INSTALLED,
                }),
                macOf(MINI, MINI_MACHINES),
            ],
            ONE_DOWN,
        ),
    },
    {
        name: 'dense fleet: 4 Macs, 8 machines each, a long name',
        props: board(['studio-1', 'studio-2', 'studio-3', 'studio-4'].map(denseMac), ALL_RUNNING, {
            foot: ADD_A_MAC,
        }),
    },
    {
        name: 'a Mac chosen',
        props: board(FLEET, ALL_RUNNING, { chosenAtFirst: 'mac-mini', foot: ADD_A_MAC }),
    },
    {
        name: 'a row opened',
        props: board(FLEET, ALL_RUNNING, { foot: ADD_A_MAC, openedAtFirst: 'linux-1' }),
    },
];
