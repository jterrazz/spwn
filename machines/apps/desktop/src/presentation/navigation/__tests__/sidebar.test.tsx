import { button, component, heading, navigation, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { composeReplay } from '../../../composition/compose';
import { recordingsOf } from '../../../infrastructure/outbound/engine-replay/recordings';
import { EngineProvider } from '../../dependencies/dependencies';
import { PAGES } from '../../screens/registry';
import { Window } from '../window';

const recordingFor = await recordingsOf('stories');

const sidebar = navigation('Sidebar');

describe('Sidebar', () => {
    test("the foot's Engine line opens the Engine page, with no release out", async () => {
        // Given - the window on Home over stories, whose engine is not behind
        const dependencies = composeReplay({ recording: recordingFor('stories') });
        const result = await component.render(
            <EngineProvider dependencies={dependencies}>
                <Window initial={{ page: 'home' }} pages={PAGES} />
            </EngineProvider>,
            async (visitor) => {
                // When - the foot's Engine line is pressed
                await visitor.click(within(sidebar, button('Engine running · v4.0.0')));

                // Then - the Engine page, its head as the engine sent it
                await visitor.see(heading('The latest release is not known yet.'));
            },
        );
        expect(result.errors.text).toBe('');
    });
});
