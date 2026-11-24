import type { MacSectionProps } from './mac-section';
import { Panel } from './panel';
import { Row } from './row';
import type { Story } from './story';

export const MAC_SECTION_STORIES: readonly Story<MacSectionProps>[] = [
    {
        name: 'a Mac and one panel',
        props: {
            children: (
                <Panel title="Engine">
                    <Row state={{ tone: 'ok', word: 'running' }} title="Engine" value="v4.0.0" />
                </Panel>
            ),
            name: 'macbook-pro',
        },
    },
    {
        name: 'a long name',
        props: {
            children: <Panel title="Disk" />,
            name: 'the-mac-mini-under-the-desk-in-the-back-office',
        },
    },
];
