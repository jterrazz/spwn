import { button, component, content, dialog, field, region, within } from '@jterrazz/test';
import { useEffect, useState } from 'react';
import type { ReactNode } from 'react';
import { describe, expect, test } from 'vitest';

import { Client } from '../../../engine/client';
import { EngineProvider } from '../../../engine/react';
import { replayTransport } from '../../../engine/replay';
import { useGesture } from '../../../engine/use-gesture';
import { pageOver } from '../../../fixtures/page-over';
import { recordingsOf } from '../../../fixtures/recordings';
import { GestureButton, GestureConfirm, GestureOutcome } from '../gestures';

const recordingFor = await recordingsOf('stories', 'two-macs');

const erasing = dialog('Erase orbstack?');
const refusal = content('has no data station can erase', { exact: false });

function Erase({ label }: { label: string }) {
    const erase = useGesture('disk.erase', {
        input: { App: 'orbstack' },
        rereads: [],
        subject: 'orbstack',
    });
    return (
        <div>
            <GestureButton
                gesture={erase}
                label={label}
                title="Erase orbstack?"
                variant="danger"
                verb="Erase"
            />
            <GestureOutcome gesture={erase} />
        </div>
    );
}

function Rebuilt() {
    const [shown, setShown] = useState(true);
    return (
        <>
            <button
                onClick={() => {
                    setShown(!shown);
                }}
                type="button"
            >
                {shown ? 'Leave' : 'Come back'}
            </button>
            {shown ? <Erase label="Erase…" /> : null}
        </>
    );
}

const rebuilds = new EventTarget();

function RebuiltWhileAsking() {
    const [generation, setGeneration] = useState(0);
    useEffect(() => {
        const next = () => {
            setGeneration((before) => before + 1);
        };
        rebuilds.addEventListener('rebuild', next);
        return () => {
            rebuilds.removeEventListener('rebuild', next);
        };
    }, []);
    return <Erase key={generation} label={generation === 0 ? 'Erase…' : 'Erase, rebuilt'} />;
}

function heightOf(panel: string): number {
    return (
        document.querySelector(`section[aria-label="${panel}"]`)?.getBoundingClientRect().height ??
        0
    );
}

function over(children: ReactNode) {
    const client = new Client(replayTransport(recordingFor('two-macs')));
    return (
        <EngineProvider client={client}>
            {children}
            <GestureConfirm />
        </EngineProvider>
    );
}

describe('gestures', () => {
    test('a row the answer changes in place carries its mark beside its button at its height, and no Done here is drawn', async () => {
        // Given - linux-1's Accounts over stories, its claude copy ending in 24 min
        let before = 0;
        let after = 0;
        const result = await component.render(
            pageOver(recordingFor('stories'), {
                machine: 'linux-1',
                page: 'machine',
                tab: 'Accounts',
            }),
            async (visitor) => {
                const shared = region('Shared with it');
                await visitor.see(within(shared, content('ends in 24 min')));
                before = heightOf('Shared with it');

                // When - Extend of claude is clicked and accounts.list read again
                await visitor.click(within(shared, button('Extend claude by 12 h')));
                await visitor.see(within(shared, content('ends in 11 h 59 min')));

                // Then - the row says done where it stands, and no panel repeats it
                await visitor.see(within(shared, content('Done')));
                await visitor.gone(region('Done here'));
                after = heightOf('Shared with it');
            },
        );
        expect(result.errors.text).toBe('');
        expect(before).toBeGreaterThan(0);
        expect(after).toBe(before);
    });

    test('a row the answer no longer lists is held in Done here', async () => {
        // Given - linux-1's Ports over two-macs, :5173 shared
        const result = await component.render(
            pageOver(recordingFor('two-macs'), {
                machine: 'linux-1',
                page: 'machine',
                tab: 'Ports',
            }),
            async (visitor) => {
                await visitor.see(region('Port 5173'));

                // When - its share is stopped and the stop confirmed
                await visitor.click(within(region('Port 5173'), button('Stop sharing')));
                const asked = dialog('Stop sharing linux-1:5173?');
                await visitor.click(within(asked, button('Stop sharing')));

                // Then - the row is gone from the list, and its outcome is held above it
                await visitor.gone(region('Port 5173'));
                await visitor.see(region('Done here'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('two buttons of one request share its state, and only the one clicked asks', async () => {
        // Given - two buttons erasing orbstack on the same Mac, each with its own hook
        const result = await component.render(
            over(
                <>
                    <Erase label="Erase in the list" />
                    <Erase label="Erase in the header" />
                </>,
            ),
            async (visitor) => {
                // When - the one in the header is clicked
                await visitor.click(button('Erase in the header'));
                await visitor.see(erasing);
            },
        );

        // Then - one dialog is open, not one per button
        expect(result.html.text.match(/role="dialog"/gu)).toHaveLength(1);
    });

    test('a refusal outlives its row leaving and coming back', async () => {
        // Given - an erase of orbstack the engine refuses
        const result = await component.render(over(<Rebuilt />), async (visitor) => {
            await visitor.click(button('Erase…'));
            await visitor.fill(within(erasing, field('Type orbstack to confirm')), 'orbstack');
            await visitor.click(within(erasing, button('Erase')));
            await visitor.see(refusal);

            // When - its row leaves the page, then is drawn again by a new component
            await visitor.click(button('Leave'));
            await visitor.gone(refusal);
            await visitor.click(button('Come back'));

            // Then - the new row shows the same refusal
            await visitor.see(refusal);
        });
        expect(result.errors.text).toBe('');
    });

    test('a dialog outlives the row that opened it being rebuilt, and still confirms', async () => {
        // Given - an erase of orbstack asking its name, the name typed
        const result = await component.render(over(<RebuiltWhileAsking />), async (visitor) => {
            await visitor.click(button('Erase…'));
            await visitor.fill(within(erasing, field('Type orbstack to confirm')), 'orbstack');

            // When - the row is rebuilt by a new component while the dialog is open
            rebuilds.dispatchEvent(new Event('rebuild'));
            await visitor.see(button('Erase, rebuilt'));

            // Then - the dialog is still open with the name, and confirming sends the erase
            await visitor.see(erasing);
            await visitor.click(within(erasing, button('Erase')));
            await visitor.gone(erasing);
            await visitor.see(refusal);
        });
        expect(result.errors.text).toBe('');
        expect(result.html.text).not.toMatch(/role="dialog"/u);
    });
});
