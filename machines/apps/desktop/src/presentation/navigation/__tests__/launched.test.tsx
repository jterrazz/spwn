import { component, content, heading, navigation, waitUntil, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import type { Link } from '../../../application/ports/engine.port';
import { composeReplay } from '../../../composition/compose';
import { recordingsOf } from '../../../infrastructure/outbound/engine-replay/recordings';
import type { Recording } from '../../../infrastructure/outbound/engine-replay/replay';
import { EngineProvider } from '../../dependencies/dependencies';
import { PAGES } from '../../screens/registry';
import { Launched } from '../launched';
import { Window } from '../window';

const recordingFor = await recordingsOf('two-macs');

const NONE: Recording = { exchanges: [], station: '', world: 'none' };

function launchedOver(recording: Recording, link?: Link) {
    const drawn: string[] = [];
    const onDrawn = () => {
        drawn.push(document.querySelector('nav[aria-label="Sidebar"]')?.textContent ?? '');
    };
    const dependencies = composeReplay({ link, recording });
    const tree = (
        <EngineProvider dependencies={dependencies}>
            <Window initial={{ page: 'home' }} pages={PAGES} />
            <Launched onDrawn={onDrawn} />
        </EngineProvider>
    );
    return { drawn, tree };
}

describe('Launched', () => {
    test("the window is drawn once machines.list's first out is in the frame", async () => {
        // Given - the window launched over two-macs
        const { drawn, tree } = launchedOver(recordingFor('two-macs'));

        // When - the engine has answered
        await component.render(tree, async (visitor) => {
            await visitor.see(
                within(navigation('Sidebar'), content('macbook-pro', { exact: false })),
            );
            await waitUntil(() => drawn.length > 0, { why: 'the frame drawn' });
        });

        // Then - the commit it is told of already holds the machines in the sidebar
        expect(drawn[0]).toContain('macbook-pro');
    });

    test('no engine answering is drawn too: the core panel is the page', async () => {
        // Given - the window launched over a socket nobody answers
        const { drawn, tree } = launchedOver(NONE, {
            kind: 'absent',
            reason: 'connection refused',
            retryIn: 4000,
            socket: '/run/engine.sock',
        });

        // When - the transport has said so
        await component.render(tree, async (visitor) => {
            await visitor.see(heading('Spwn is not installed on this Mac.'));
            await waitUntil(() => drawn.length > 0, { why: 'the frame drawn' });
        });

        // Then - told once the panel stands
        expect(drawn.length).toBeGreaterThan(0);
    });

    test('a link still connecting is not data: the cap shows it', async () => {
        // Given - the window launched over a link still connecting
        const { drawn, tree } = launchedOver(NONE, {
            kind: 'connecting',
            socket: '/run/engine.sock',
        });

        // When - the frame says the engine is connecting
        await component.render(tree, async (visitor) => {
            await visitor.see(within(navigation('Sidebar'), content('Engine connecting')));
        });

        // Then - nothing is told
        expect(drawn).toStrictEqual([]);
    });
});
