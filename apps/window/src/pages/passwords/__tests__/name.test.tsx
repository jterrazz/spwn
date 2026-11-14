import { component, content, heading } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { Client } from '../../../engine/client';
import { EngineProvider } from '../../../engine/react';
import { replayTransport } from '../../../engine/replay';
import { recordingsOf } from '../../../fixtures/recordings';
import { Window } from '../../../shell/window';
import { PAGES } from '../../registry';

const recordingFor = await recordingsOf('two-macs');

describe('name', () => {
    test("a password's crumb is its label, never its id", async () => {
        // Given - the window opened on npm publish, whose id is pw-2
        const client = new Client(replayTransport(recordingFor('two-macs')));
        const result = await component.render(
            <EngineProvider client={client}>
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
