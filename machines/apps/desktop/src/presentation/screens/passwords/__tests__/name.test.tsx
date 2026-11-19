import { component, content, heading } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { composeReplay } from '../../../../composition/compose';
import { recordingsOf } from '../../../../infrastructure/outbound/engine-replay/recordings';
import { EngineProvider } from '../../../dependencies/dependencies';
import { Window } from '../../../navigation/window';
import { PAGES } from '../../registry';

const recordingFor = await recordingsOf('two-macs');

describe('name', () => {
    test("a password's crumb is its label, never its id", async () => {
        // Given - the window opened on npm publish, whose id is pw-2
        const dependencies = composeReplay({ recording: recordingFor('two-macs') });
        const result = await component.render(
            <EngineProvider dependencies={dependencies}>
                <Window initial={{ item: 'pw-2', page: 'passwords' }} pages={PAGES} />
            </EngineProvider>,
            async (visitor) => {
                // When - the vault has answered
                await visitor.see(heading('npm publish'));

                // Then - the toolbar names it by its label
                await visitor.see(content('Passwords › npm publish', { exact: false }));
            },
        );
        expect(result.errors.text).toBe('');
        expect(result.content.text).not.toContain('pw-2');
        expect(result.content.text).not.toContain('Add a password');
    });
});
