import '../../ui/document.css';
import { banner, button, component, content, dialog, field, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { composeReplay } from '../../../composition/compose';
import { recordingsOf } from '../../../infrastructure/outbound/engine-replay/recordings';
import { EngineProvider } from '../../dependencies/dependencies';
import { PAGES } from '../../screens/registry';
import { Window } from '../window';

const recordingFor = await recordingsOf('two-macs');

// hazard: press wraps a key longer than one letter in braces, so a chord is user-event's hold and release
// hazard: with its outer braces left to press.
const COMMAND_K = 'Control>}k{/Control';

const find = dialog('Find');
const query = within(find, field('Find'));
const FIND = within(banner(), button('Find'));

function windowOnHome() {
    return (
        <EngineProvider dependencies={composeReplay({ recording: recordingFor('two-macs') })}>
            <Window initial={{ page: 'home' }} pages={PAGES} />
        </EngineProvider>
    );
}

function shown(part: Element | null): boolean {
    return part !== null && getComputedStyle(part).display !== 'none';
}

// value: the bar's Find as drawn: its width, its fill, and whether its word and its ⌘K hint show.
function drawnFind() {
    const found = document.querySelector<HTMLButtonElement>('header button[aria-label="Find"]');
    if (found === null) {
        throw new Error('the bar draws no Find');
    }
    return {
        filled: getComputedStyle(found).backgroundColor !== 'rgba(0, 0, 0, 0)',
        hint: shown(found.querySelector('kbd')),
        width: Math.round(found.getBoundingClientRect().width),
        word: shown(found.querySelector('span')),
    };
}

describe('PathBar', () => {
    test('under 1280 px Find is its glass alone, 28 px, and both its press and ⌘K open the same Find', async () => {
        // Given - Home over two-macs in a 1100 px window, the window's minimum
        let drawn = {};
        await component
            .viewport({ height: 700, width: 1100 })
            .render(windowOnHome(), async (visitor) => {
                await visitor.see(content('2 things need you.'));
                drawn = drawnFind();

                // When - the glass is pressed, then Find closed and ⌘K pressed
                await visitor.click(FIND);
                await visitor.see(query);
                await visitor.press('Escape');
                await visitor.gone(find);
                await visitor.press(COMMAND_K);

                // Then - Find opens both times
                await visitor.see(query);
            });

        // Then - the glass alone and bare, named Find
        expect(drawn).toStrictEqual({ filled: false, hint: false, width: 28, word: false });
    });

    test('from 1280 px Find keeps its word and its ⌘K hint, 120 px wide', async () => {
        // Given - Home over two-macs in a 1440 px window
        let drawn = {};
        await component
            .viewport({ height: 900, width: 1440 })
            .render(windowOnHome(), async (visitor) => {
                await visitor.see(content('2 things need you.'));
                drawn = drawnFind();

                // When - ⌘K
                await visitor.press(COMMAND_K);

                // Then - the same Find opens
                await visitor.see(query);
            });

        // Then - the field as before, filled, its word and hint drawn
        expect(drawn).toStrictEqual({ filled: true, hint: true, width: 120, word: true });
    });
});
