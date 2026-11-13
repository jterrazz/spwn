import { Button } from './button';
import { Caption } from './caption';
import { Missing } from './missing';
import type { PanelProps } from './panel';
import { Row } from './row';
import type { Story } from './story';

export const PANEL_STORIES: readonly Story<PanelProps>[] = [
    {
        name: 'rows, a note and an action',
        props: {
            action: <Button variant="link">3 more can wait</Button>,
            children: (
                <>
                    <Row
                        sentence="jterrazz-macbook-pro · AirPlay Receiver"
                        state={{ tone: 'bad', word: 'Exposed' }}
                        stateAt="start"
                        title="Port 5000 is reachable from your local network"
                    />
                    <Row size="slim" title="node 24.11.0 → 24.11.1" />
                </>
            ),
            note: '2',
            title: 'Needs you',
        },
    },
    {
        name: 'padded, a caption',
        props: {
            children: <Caption>A caption heads a group of rows</Caption>,
            padded: true,
            title: 'Padded',
        },
    },
    {
        name: 'missing',
        props: {
            children: <Missing>What no call answers is named here, as the field it lacks.</Missing>,
            title: 'A field no call gives',
        },
    },
    { name: 'empty', props: { note: 'it says so', title: 'An empty panel' } },
    {
        name: 'long text',
        props: {
            note: 'a note long enough to share the head with a title that runs on as well',
            title: 'A panel whose title is long enough to wrap beside its note',
        },
    },
];
