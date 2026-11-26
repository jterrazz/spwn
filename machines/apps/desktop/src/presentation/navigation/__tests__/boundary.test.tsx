import { button, component, content, heading } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { composeReplay } from '../../../composition/compose';
import { recordingsOf } from '../../../infrastructure/outbound/engine-replay/recordings';
import { EngineProvider } from '../../dependencies/dependencies';
import { PAGES } from '../../screens/registry';
import type { PageProps, Pages } from '../window';
import { Window } from '../window';

const recordingFor = await recordingsOf('two-macs');

function Body({ route }: PageProps) {
    if (route.page === 'inbox') {
        throw new Error('needs is not iterable');
    }
    return <p>Home is drawn</p>;
}

const THROWING: Pages = { ...PAGES, Body, Head: () => null };

describe('Boundary', () => {
    test('a page that throws shows what broke in its frame, and the window around it survives', async () => {
        // Given - the window over two-macs, whose Inbox page throws when it renders
        const dependencies = composeReplay({ recording: recordingFor('two-macs') });
        const result = await component.render(
            <EngineProvider dependencies={dependencies}>
                <Window initial={{ page: 'home' }} pages={THROWING} />
            </EngineProvider>,
            async (visitor) => {
                await visitor.see(content('Home is drawn'));

                // When - Inbox is opened
                await visitor.click(button('Inbox 2'));

                // Then - its frame says what broke and where, with a Reload
                await visitor.see(heading('The window could not draw Inbox.'));
                await visitor.see(content('needs is not iterable'));
                await visitor.see(button('Reload'));

                // When - Home is opened from the bar that survived
                await visitor.click(button('Home'));

                // Then - Home is drawn again
                await visitor.see(content('Home is drawn'));
            },
        );
        expect(result.errors.text).toContain('needs is not iterable');
    });
});
