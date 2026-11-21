import { component, content } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { PAGE_REVISION } from '../../../domain/fit';
import { keyOf } from '../../../domain/request-key';
import { documentPresence } from '../../../infrastructure/outbound/document-presence/document-presence';
import { Client } from '../../../infrastructure/outbound/engine-link/client';
import { recordingsOf } from '../../../infrastructure/outbound/engine-replay/recordings';
import { stagedReplay } from '../../../infrastructure/outbound/engine-replay/replay';
import { NO_OPENER } from '../../../infrastructure/outbound/opener-tauri/tauri-opener';
import { EngineProvider, useRead } from '../dependencies';
import { useGesture } from '../use-gesture';

const recordingFor = await recordingsOf('two-macs');

const HELLO = { page: PAGE_REVISION, pid: 41, started_at: '2026-11-13T09:00:00Z', version: 'dev' };

function Newest() {
    const journal = useRead('journal.read', {});
    const [newest] = journal.kind === 'read' ? (journal.out.Groups ?? []) : [];
    if (newest === undefined) {
        return <p>newest: none</p>;
    }
    const lines = (newest.Lines ?? []).map((line) => line.Text).join(', ');
    return <p>{`newest: ${newest.Title}, ${lines}`}</p>;
}

function Update() {
    const gesture = useGesture('apps.update', {
        input: { Managers: ['mise'] },
        rereads: ['apps.list'],
        subject: 'mise',
    });
    return (
        <button onClick={gesture.start} type="button">
            {`Update: ${gesture.state.kind}`}
        </button>
    );
}

describe('use-gesture', () => {
    test("a gesture done reads the journal again, so its own line shows though its rereads don't name it", async () => {
        // Given - an update of mise over two-macs, whose rereads name apps.list alone, beside today's journal
        const stage = stagedReplay(recordingFor('two-macs'), {
            hello: HELLO,
            kind: 'open',
            refused: undefined,
            socket: 'replay',
        });
        const dependencies = {
            engine: new Client(stage.transport),
            opener: NO_OPENER,
            presence: documentPresence(),
        };

        // When - it is pressed and the engine answers
        const result = await component.render(
            <EngineProvider dependencies={dependencies}>
                <Update />
                <Newest />
            </EngineProvider>,
            async (visitor) => {
                await visitor.see(content('newest: Last night, mise, node'));
                await visitor.click(content('Update: idle'));
                await visitor.see(content('Update: done'));

                // Then - the newest group is today's, holding the engine's line of the update, the journal asked once more
                await visitor.see(content('newest: Today, node updated'));
            },
        );
        const asked = stage
            .asked()
            .filter((request) => keyOf(request) === 'call journal.read {} {}');
        expect(asked).toHaveLength(2);
        expect(result.errors.text).toBe('');
    });
});
