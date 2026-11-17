import { button, component, content, heading, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../../composition/__tests__/page-over';
import { recordingsOf } from '../../../../infrastructure/outbound/engine-replay/recordings';

const recordingFor = await recordingsOf('bare-mac', 'creating', 'mini-offline', 'stories');

const machines = region('Machines');

function homeOver(world: string) {
    return pageOver(recordingFor(world), { page: 'home' });
}

type Board = {
    cards: (null | string)[];
    pressed: (null | string)[];
    lines: string[][];
    expanded: (null | string)[];
    foot: string;
};

function cellsOf(opener: Element): string[] {
    const line = opener.parentElement?.parentElement;
    return [...(line?.children ?? [])].map((cell) => cell.textContent);
}

function board(): Board {
    const cards = [...document.querySelectorAll('[aria-pressed]')];
    const openers = [...document.querySelectorAll('[aria-expanded]')];
    const stage = document.querySelector('[role="group"][aria-label="Macs"]');
    return {
        cards: cards.map((card) => card.getAttribute('aria-label')),
        expanded: openers.map((opener) => opener.getAttribute('aria-expanded')),
        foot: stage?.lastElementChild?.textContent ?? '',
        lines: openers.map(cellsOf),
        pressed: cards.map((card) => card.getAttribute('aria-pressed')),
    };
}

const STORIES_CARDS = [
    'macbook-pro · 4 machines · online',
    'mac-mini · 3 machines · online',
    'macbook-air · 0 machines · asleep',
];

describe('home-fleet', () => {
    test("a Mac chosen keeps its machines' lines alone, and chosen again shows the whole fleet", async () => {
        // Given - Home over stories: macbook-pro holds four machines, mac-mini three, macbook-air none
        const states: Board[] = [];
        const result = await component.render(homeOver('stories'), async (visitor) => {
            await visitor.see(within(machines, button('review-bot')));
            states.push(board());

            // When - the mini's card is pressed, then pressed again
            await visitor.click(button(STORIES_CARDS[1] ?? ''));
            await visitor.gone(within(machines, button('agent-7f')));
            states.push(board());
            await visitor.click(button(STORIES_CARDS[1] ?? ''));
            await visitor.see(within(machines, button('agent-7f')));
            states.push(board());
        });

        // Then - the mini's three lines alone while it is pressed, the seven before and after
        const every = ['agent-7f', 'api', 'linux-1', 'linux-2', 'infra', 'linux-3', 'review-bot'];
        expect(states.map((state) => state.lines.map((cells) => cells[0]))).toStrictEqual([
            every,
            ['infra', 'linux-3', 'review-bot'],
            every,
        ]);
        expect(states.map((state) => state.pressed)).toStrictEqual([
            ['false', 'false', 'false'],
            ['false', 'true', 'false'],
            ['false', 'false', 'false'],
        ]);
        expect(states[0]?.cards).toStrictEqual(STORIES_CARDS);
        expect(states[0]?.foot).toBe('7 machinesThis Mac');
        expect(result.errors.text).toBe('');
    });

    test("a machine's line opens on the engine's sentence, and its Open goes to that machine", async () => {
        // Given - Home over stories, where infra's head says it is down while its state is running
        const states: Board[] = [];
        const result = await component.render(homeOver('stories'), async (visitor) => {
            await visitor.see(within(machines, content('10 %')));

            // When - infra's line is opened, then its Open pressed, which leads to infra's own page and its head
            await visitor.click(within(machines, button('infra')));
            await visitor.see(within(machines, content('infra is down.')));
            await visitor.see(within(machines, content('Its check fails since 10:18.')));
            states.push(board());
            await visitor.click(within(machines, button('Open infra')));
            await visitor.see(heading('infra is down.'));
        });

        // Then - infra alone was expanded, its line reading its purpose, its state, its CPU and its memory
        expect(states[0]?.expanded).toStrictEqual([
            'false',
            'false',
            'false',
            'false',
            'true',
            'false',
            'false',
        ]);
        expect(states[0]?.lines[4]).toStrictEqual([
            'infra',
            "The cluster's control plane",
            'Running',
            '10 %',
            '',
            '16 / 24',
        ]);
        expect(result.errors.text).toBe('');
    });

    test("a Mac's name on its line goes to the Mac", async () => {
        // Given - Home over stories
        const result = await component.render(homeOver('stories'), async (visitor) => {
            await visitor.see(within(machines, button('mac-mini')));

            // When - the mini's name is pressed on its line
            await visitor.click(within(machines, button('mac-mini')));

            // Then - the mini's page lists its machines
            await visitor.see(heading('Its machines'));
        });
        expect(result.content.text).toContain('review-bot');
        expect(result.errors.text).toBe('');
    });

    test('a Mac not seen is drawn by its state, its machines by theirs, with no figure the engine did not measure', async () => {
        // Given - Home over mini-offline: mac-mini does not answer, infra on it is not seen, and machines.list gives no room
        const states: Board[] = [];
        const result = await component.render(homeOver('mini-offline'), async (visitor) => {
            // When - the fleet is drawn
            await visitor.see(within(machines, button('infra')));
            states.push(board());
        });

        // Then - each card says its Mac's state, and no line draws memory nor a track
        expect(states[0]?.cards).toStrictEqual([
            'macbook-pro · 2 machines · online',
            'mac-mini · 1 machine · not seen',
        ]);
        expect(states[0]?.lines.map((cells) => [cells[0], cells[2], cells[5]])).toStrictEqual([
            ['api', 'Stopped', ''],
            ['linux-1', 'Running', ''],
            ['infra', 'Not seen', ''],
        ]);
        expect(result.html.text).not.toContain('Memory of');
        expect(result.errors.text).toBe('');
    });

    test('a machine being created reads Creating, and its share holds no use yet', async () => {
        // Given - Home over creating: linux-4 (8 GB) and linux-5 (4 GB) being created on a 24 GB Mac, 8 free
        const states: Board[] = [];
        const result = await component.render(homeOver('creating'), async (visitor) => {
            // When - the fleet is drawn
            await visitor.see(within(machines, button('linux-5')));
            states.push(board());
        });

        // Then - both read the engine's Creating, and their figure is their share alone
        expect(states[0]?.lines.map((cells) => [cells[0], cells[2], cells[5]])).toStrictEqual([
            ['linux-4', 'Creating', 'of 8'],
            ['linux-5', 'Creating', 'of 4'],
        ]);
        expect(result.content.text).toContain('8 free');
        expect(result.errors.text).toBe('');
    });

    test('a bare Mac is one card with no machine, and the stage counts none', async () => {
        // Given - Home over bare-mac, whose one Mac lists no machine
        const states: Board[] = [];
        const result = await component.render(homeOver('bare-mac'), async (visitor) => {
            // When - the fleet is drawn
            await visitor.see(button('macbook-pro · 0 machines · online'));
            states.push(board());
        });

        // Then - its card, no line to open, and the stage's count of none
        expect(states).toStrictEqual([
            {
                cards: ['macbook-pro · 0 machines · online'],
                expanded: [],
                foot: '0 machinesThis Mac',
                lines: [],
                pressed: ['false'],
            },
        ]);
        expect(result.errors.text).toBe('');
    });
});
