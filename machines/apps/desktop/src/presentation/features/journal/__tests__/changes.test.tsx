import { button, component, content, file, heading, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../../composition/__tests__/page-over';
import { recordingsOf } from '../../../../infrastructure/outbound/engine-replay/recordings';

const recordingFor = await recordingsOf('stories');

const history = region('History');
const lastNight = region('Last night');

describe('changes', () => {
    test("the Inbox's History draws the engine's days, the night's failure first with its Retry", async () => {
        // Given - the Inbox over stories, whose night moved three packages, held one and failed one
        const result = await component.render(
            pageOver(recordingFor('stories'), { page: 'inbox' }),
            async (visitor) => {
                // When - the journal has answered
                await visitor.see(within(history, heading('Last night 3 moved, 1 held, 1 failed')));
            },
        );

        // Then - Today, Last night and Yesterday, each line with its time, who, what moved and on how many machines
        await expect(result.tree).toMatch(file('./__expected__/stories-history.aria.yaml'));

        // And - a held line draws the engine's word as a plain tag, a failed one as a pill
        expect(result.html.text).toMatch(/<span class="[^"]*_end_[^"]*">held</u);
        expect(result.html.text).not.toMatch(/_pill_[^>]*>held</u);
        expect(result.html.text).toMatch(/_pill_[^>]*>failed</u);
    });

    test('home draws the last night alone, with its first three lines', async () => {
        // Given - Home over stories
        const result = await component.render(
            pageOver(recordingFor('stories'), { page: 'home' }),
            async (visitor) => {
                // When - the night has answered
                await visitor.see(within(lastNight, button('Retry ripgrep')));
            },
        );

        // Then - the failure and the two lines after it, and neither Today nor Yesterday
        await expect(result.tree).toMatch(file('./__expected__/stories-home.aria.yaml'));
    });

    test("choosing a Mac draws the engine's reading of that Mac alone, with no day's sum", async () => {
        // Given - the Inbox over stories
        const result = await component.render(
            pageOver(recordingFor('stories'), { page: 'inbox' }),
            async (visitor) => {
                await visitor.see(within(history, heading('Today')));

                // When - mac-mini is chosen above the list
                await visitor.click(button('mac-mini'));

                // Then - gh, kept by macbook-pro alone, has left
                await visitor.see(within(history, heading('Last night')));
                await visitor.gone(within(history, content('gh')));
            },
        );
        await expect(result.tree).toMatch(
            file('./__expected__/stories-history-mac-mini.aria.yaml'),
        );
    });
});
