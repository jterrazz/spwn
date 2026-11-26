import { component, content, navigation } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { composeReplay } from '../../../composition/compose';
import { recordingsOf } from '../../../infrastructure/outbound/engine-replay/recordings';
import { EngineProvider } from '../../dependencies/dependencies';
import { PAGES } from '../../screens/registry';
import { Window } from '../window';

const recordingFor = await recordingsOf('two-macs');

const column = navigation('Column');

describe('column', () => {
    test.each([
        { pinned: true, width: 1456 },
        { pinned: false, width: 1280 },
    ])(
        'a window $width px wide that never chose stands its column: $pinned',
        async ({ pinned, width }) => {
            // Given - Home over two-macs, no choice kept, the window $width px wide
            const result = await component.viewport({ height: 900, width }).render(
                <EngineProvider
                    dependencies={composeReplay({ recording: recordingFor('two-macs') })}
                >
                    <Window initial={{ page: 'home' }} pages={PAGES} />
                </EngineProvider>,
                async (visitor) => {
                    // When - Home has been read
                    await visitor.see(content('2 things need you.'));

                    // Then - the column stands from 1440 px, and hides under it
                    await (pinned ? visitor.see(column) : visitor.gone(column));
                },
            );
            expect(result.errors.text).toBe('');
        },
    );
});
