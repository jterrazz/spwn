import type { GuideProps, GuideStep, GuideStepStatus } from './guide';
import { nothing } from './story';
import type { Story } from './story';

function stepsOf(
    labels: readonly string[],
    at: number,
    status: GuideStepStatus = 'now',
    detail?: string,
): GuideStep[] {
    return labels.map((label, index) => {
        if (index < at) {
            return { id: label, label, status: 'done' };
        }
        if (index === at) {
            return { detail, id: label, label, status };
        }
        return { id: label, label, status: 'next' };
    });
}

const CREATE = ['Get the image', 'Build it', 'Boot', 'Apply its file', 'Ready'];
const ADD_A_MAC = ['Reach mac-mini', 'Sign as this Mac', 'Install Spwn there', 'Joined'];
const FIRST_LAUNCH = ['Install the engine', 'Allow Local Network', 'Sign in a program', 'Ready'];
const SEVEN = [
    'Install the engine',
    'Allow Local Network',
    'Allow notifications',
    'Sign in to GitHub',
    'Sign in to Claude',
    'First machine',
    'Ready',
];
const TEN = [
    'Install the engine',
    'Allow Local Network',
    'Allow notifications',
    'Sign in to GitHub',
    'Sign in to Claude',
    'Add mac-mini',
    'Add macbook-air',
    'First machine',
    'Its file',
    'Ready',
];
const KEEPS_GOING = { label: 'Hide, it keeps going', onPress: nothing };

const NEW_MACHINE = { act: 'New machine', onHide: nothing, subject: 'linux-5' };

export const GUIDE_STORIES: readonly Story<GuideProps>[] = [
    {
        name: 'working, measured',
        props: {
            ...NEW_MACHINE,
            steps: stepsOf(CREATE, 1, 'now', 'Packages · 2 of 4'),
            view: {
                action: KEEPS_GOING,
                headline: { next: 'Packages, 2 of 4.', now: 'Building the image.' },
                illustration: 'build-2',
                move: 'engine',
                progress: { done: 214, eta: 'about 40 s', kind: 'measure', total: 380, unit: 'MB' },
                sentence:
                    'The first machine of an image takes longest. The next ones start from it in seconds.',
            },
        },
    },
    {
        name: 'working, its last line',
        props: {
            ...NEW_MACHINE,
            steps: stepsOf(CREATE, 2),
            view: {
                action: KEEPS_GOING,
                headline: { next: 'Waiting for its address.', now: 'Booting linux-5.' },
                illustration: 'boot',
                move: 'engine',
                progress: { kind: 'live', line: 'asked for an address 9 s ago' },
                sentence:
                    'The machine is up. Spwn waits for it to answer on the network, usually under a minute.',
            },
        },
    },
    {
        name: 'waiting on you, here',
        props: {
            act: 'Add a Mac',
            onHide: nothing,
            steps: stepsOf(ADD_A_MAC, 1, 'waiting'),
            subject: 'mac-mini',
            view: {
                action: { label: 'Sign with Touch ID →', onPress: nothing },
                headline: { next: 'Sign once to reach mac-mini.', now: 'Touch ID is next.' },
                illustration: 'touch-id',
                move: 'owner',
                prompt: {
                    lead: 'macOS will ask:',
                    quote: 'Spwn Engine wants to sign in over ssh as this Mac.',
                },
                quiet: { label: 'Not now', onPress: nothing },
                sentence:
                    'This Mac signs in to mac-mini with its own key. Your fingerprint never leaves this Mac.',
            },
        },
    },
    {
        name: 'waiting on you, elsewhere',
        props: {
            act: 'Set up this Mac',
            onHide: nothing,
            steps: stepsOf(FIRST_LAUNCH, 1, 'waiting'),
            subject: 'macbook-pro',
            view: {
                action: { label: 'Open System Settings ↗', onPress: nothing },
                headline: {
                    next: 'Turn it on in System Settings.',
                    now: 'Spwn needs Local Network.',
                },
                illustration: 'system-settings',
                move: 'owner',
                progress: { kind: 'live', line: 'watching for the switch' },
                quiet: { label: 'Skip', onPress: nothing },
                sentence:
                    'Without it, this Mac cannot see your other Macs. The guide moves on by itself once it is on.',
            },
        },
    },
    {
        name: 'failed',
        props: {
            ...NEW_MACHINE,
            steps: stepsOf(CREATE, 2, 'failed'),
            view: {
                action: { label: 'Try the boot again →', onPress: nothing },
                headline: { next: 'Try the boot again.', now: 'linux-5 did not boot.' },
                illustration: 'boot-failed',
                kept: 'The image and the disk are kept: a retry starts at the boot.',
                move: 'owner',
                quiet: { label: 'See the log', onPress: nothing },
                sentence: 'It gave no answer on the network in 3 min.',
            },
        },
    },
    {
        name: 'done',
        props: {
            act: 'New machine',
            ended: 'Done in 3 min 40 s',
            steps: stepsOf(CREATE, CREATE.length),
            subject: 'linux-5',
            view: {
                action: { label: 'Open in Terminal →', onPress: nothing },
                headline: { next: 'Open it in your terminal.', now: 'linux-5 is ready.' },
                illustration: 'done',
                move: 'owner',
                quiet: { label: 'Done', onPress: nothing },
                sentence:
                    'Ubuntu 24.04, 11 apps from its file, 4 GB of memory. It starts again with this Mac.',
            },
        },
    },
    {
        name: 'you may leave',
        props: {
            ...NEW_MACHINE,
            steps: stepsOf(CREATE, 0),
            view: {
                action: KEEPS_GOING,
                headline: { next: 'Ubuntu 24.04, from the catalog.', now: 'Getting the image.' },
                illustration: 'fetch-image',
                move: 'engine',
                progress: {
                    done: 1.1,
                    eta: 'about 2 min',
                    kind: 'measure',
                    total: 3.4,
                    unit: 'GB',
                },
                sentence: 'Fetched once for this Mac. Every next linux machine starts from it.',
            },
        },
    },
    {
        name: 'two steps',
        props: {
            act: 'Update Spwn',
            onHide: nothing,
            steps: stepsOf(['Download 0.43', 'Restart the engine'], 0, 'now', '31 of 52 MB'),
            subject: '0.42 → 0.43',
            view: {
                action: KEEPS_GOING,
                headline: { next: 'Your machines keep running.', now: 'Downloading 0.43.' },
                illustration: 'install-engine',
                move: 'engine',
                progress: { done: 31, eta: 'about 10 s', kind: 'measure', total: 52, unit: 'MB' },
                sentence: 'The engine restarts once it is here. Nothing you run stops.',
            },
        },
    },
    {
        name: 'seven steps',
        props: {
            act: 'Set up this Mac',
            onHide: nothing,
            steps: stepsOf(SEVEN, 3, 'waiting'),
            subject: 'macbook-pro',
            view: {
                action: { label: 'Sign in to GitHub ↗', onPress: nothing },
                headline: { next: 'Finish it in your browser.', now: 'GitHub is next.' },
                illustration: 'device-key',
                move: 'owner',
                progress: { kind: 'live', line: 'waiting for the browser' },
                quiet: { label: 'Skip', onPress: nothing },
                sentence:
                    'Your machines use this sign-in to clone and push. The guide moves on once you are in.',
            },
        },
    },
    {
        name: 'past seven, the done steps fold',
        props: {
            act: 'Set up this Mac',
            onHide: nothing,
            steps: stepsOf(TEN, 6, 'now'),
            subject: 'macbook-pro',
            view: {
                action: KEEPS_GOING,
                headline: {
                    next: 'Nothing to do over there.',
                    now: 'Installing Spwn on macbook-air.',
                },
                illustration: 'mac-link',
                move: 'engine',
                progress: { done: 62, eta: 'about 15 s', kind: 'measure', total: 88, unit: 'MB' },
                sentence: 'The same version as this Mac. macbook-air keeps running what it runs.',
            },
        },
    },
];
