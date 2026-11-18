import type { RadioRowsProps } from './radio-rows';
import { LONG_NAME, nothing } from './story';
import type { Story } from './story';

const ROWS = [
    { id: 'linux-1', line: 'Signed out here', title: 'linux-1' },
    { held: true, id: 'api', line: 'Start the machine first', title: 'api' },
    { held: true, id: 'infra', line: 'A server keeps its own sign-ins', title: 'infra' },
];

export const RADIO_ROWS_STORIES: readonly Story<RadioRowsProps<string>>[] = [
    {
        name: 'nothing chosen, two held with their reason',
        props: { label: 'Machines', onChange: nothing, rows: ROWS, value: null },
    },
    {
        name: 'one chosen',
        props: { label: 'Machines', onChange: nothing, rows: ROWS, value: 'linux-1' },
    },
    {
        name: 'every row held',
        props: {
            label: 'Machines',
            onChange: nothing,
            rows: ROWS.map((row) => ({ ...row, held: true })),
            value: null,
        },
    },
    {
        name: 'a long name and a long reason',
        props: {
            label: 'Machines',
            onChange: nothing,
            rows: [
                { id: 'long', line: 'Signed out here', title: LONG_NAME },
                {
                    held: true,
                    id: 'mini',
                    line: 'Its Mac does not answer, so nothing says whether it can take a password',
                    title: 'mac-mini-vm',
                },
            ],
            value: null,
        },
    },
];
