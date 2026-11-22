import {
    button,
    component,
    content,
    dialog,
    field,
    heading,
    navigation,
    within,
} from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import type { Link } from '../../../application/ports/engine.port';
import { documentPresence } from '../../../infrastructure/outbound/document-presence/document-presence';
import { Client } from '../../../infrastructure/outbound/engine-link/client';
import { helloOf } from '../../../infrastructure/outbound/engine-replay/hello';
import { recordingsOf } from '../../../infrastructure/outbound/engine-replay/recordings';
import { stagedReplay } from '../../../infrastructure/outbound/engine-replay/replay';
import type { Stage } from '../../../infrastructure/outbound/engine-replay/replay';
import { NO_OPENER } from '../../../infrastructure/outbound/opener-tauri/tauri-opener';
import { EngineProvider } from '../../dependencies/dependencies';
import { PAGES } from '../../screens/registry';
import { Window } from '../window';

const recordingFor = await recordingsOf('two-macs');

const sidebar = navigation('Sidebar');
const rail = navigation('Rail');
const SOCKET = 'replay of two-macs';
const ABSENT: Link = {
    kind: 'absent',
    reason: 'connection refused',
    retryIn: 1000,
    socket: SOCKET,
};
const OPEN: Link = {
    hello: helloOf(recordingFor('two-macs')),
    kind: 'open',
    refused: undefined,
    socket: SOCKET,
};
const SAID = within(rail, content(`No engine answers at ${SOCKET}`));
const VERDICT = content('2 things need you.');
const FIND = dialog('Find');

function windowOver(stage: Stage) {
    const dependencies = {
        engine: new Client(stage.transport),
        opener: NO_OPENER,
        presence: documentPresence(),
    };
    return (
        <EngineProvider dependencies={dependencies}>
            <Window initial={{ page: 'home' }} pages={PAGES} />
        </EngineProvider>
    );
}

function pageRoot(): Element | null | undefined {
    return document.querySelector('main > [role="none"] + div')?.firstElementChild;
}

describe('no engine', () => {
    test('an absence over drawn pages keeps the page and says it in the foot, and the relink remounts nothing', async () => {
        // Given - Home drawn over two-macs
        const stage = stagedReplay(recordingFor('two-macs'));
        const roots: (Element | null | undefined)[] = [];
        const result = await component.render(windowOver(stage), async (visitor) => {
            await visitor.see(VERDICT);
            roots.push(pageRoot());

            // When - the link is declared absent, then opens again on the same engine
            stage.show(ABSENT);
            await visitor.see(SAID);
            await visitor.see(VERDICT);
            roots.push(pageRoot());
            stage.show(OPEN);
            await visitor.gone(SAID);
            await visitor.see(VERDICT);
            roots.push(pageRoot());
        });

        // Then - one page root through all three, and no no-engine panel
        expect(roots[0]).toBeInstanceOf(HTMLElement);
        expect(roots[1]).toBe(roots[0]);
        expect(roots[2]).toBe(roots[0]);
        expect(result.html.text).not.toContain('Spwn is not installed on this Mac.');
        expect(result.errors.text).toBe('');
    });

    test('an open sheet keeps its draft through an absence and a relink', async () => {
        // Given - Find opened over drawn pages, a query typed in it
        const stage = stagedReplay(recordingFor('two-macs'));
        const query = within(FIND, field('Find'));
        let draft: string | undefined;
        const result = await component.render(windowOver(stage), async (visitor) => {
            await visitor.see(VERDICT);
            await visitor.click(within(sidebar, button('Find')));
            await visitor.fill(query, 'linux');

            // When - the link is declared absent, then opens again
            stage.show(ABSENT);
            await visitor.see(SAID);
            stage.show(OPEN);
            await visitor.gone(SAID);

            // Then - the sheet is still open, its field holding the query
            await visitor.see(FIND);
            draft = document.querySelector<HTMLInputElement>('[role="dialog"] input')?.value;
        });
        expect(draft).toBe('linux');
        expect(result.errors.text).toBe('');
    });

    test('an absence before anything was drawn is the panel, and the engine answering draws Home in the same frame', async () => {
        // Given - a link declared absent from the start, and no core on the disk
        const stage = stagedReplay(recordingFor('two-macs'), ABSENT);
        const mains: (Element | null)[] = [];
        const result = await component.render(windowOver(stage), async (visitor) => {
            await visitor.see(heading('Spwn is not installed on this Mac.'));
            mains.push(document.querySelector('main'));

            // When - the engine answers
            stage.show(OPEN);

            // Then - Home is drawn in the main the panel stood in
            await visitor.see(VERDICT);
            mains.push(document.querySelector('main'));
        });
        expect(mains[0]).toBeInstanceOf(HTMLElement);
        expect(mains[1]).toBe(mains[0]);
        expect(result.errors.text).toBe('');
    });
});
