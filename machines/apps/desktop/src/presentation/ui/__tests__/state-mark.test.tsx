import { component } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { TONES } from '../../../domain/tone';

import '../document.css';
import { StateMark } from '../state-mark';

function marks(): HTMLElement[] {
    return [...document.querySelectorAll<HTMLElement>('[data-mark]')].filter(
        (one) => one.textContent === '',
    );
}

function shapeOf(mark: HTMLElement): string {
    const style = getComputedStyle(mark);
    return [
        style.width,
        style.height,
        style.borderRadius,
        style.transform,
        style.backgroundColor === 'rgba(0, 0, 0, 0)' ? 'hollow' : 'filled',
        style.backgroundImage === 'none' ? 'flat' : 'cut',
        style.boxShadow === 'none' ? 'bare' : 'ringed',
    ].join(' ');
}

function playStateOf(visibility: 'hidden' | 'visible'): string {
    document.documentElement.dataset.visibility = visibility;
    const busy = document.querySelector('[data-mark="busy"]');
    return busy === null ? '' : getComputedStyle(busy).animationPlayState;
}

describe('StateMark', () => {
    test('each tone draws a shape of its own, so a state reads without its colour, and each says its word', async () => {
        // Given - the five tones, each named by its word
        let shapes: string[] = [];
        const result = await component.render(
            <div>
                {TONES.map((tone) => (
                    <StateMark key={tone} said="name" tone={tone} word={`word of ${tone}`} />
                ))}
            </div>,
            async () => {
                // When - their marks are measured
                shapes = marks().map(shapeOf);
            },
        );

        // Then - five marks, five shapes, five names
        expect(shapes).toHaveLength(TONES.length);
        expect(new Set(shapes).size).toBe(TONES.length);
        expect(result.tree.text).toBe(TONES.map((tone) => `- img "word of ${tone}"`).join('\n'));
    });

    test('a calm state draws no word: its mark carries the word as its name', async () => {
        // Given - a machine that runs
        const result = await component.render(<StateMark tone="ok" word="running" />);

        // Then - no text is drawn, and the mark is an image named by the word
        expect(result.content.text).toBe('');
        expect(result.tree.text).toBe('- img "running"');
        expect(result.html.text).toContain('title="running"');
    });

    test('a state other than calm draws its word once, and hides its mark from the reader', async () => {
        // Given - a machine down, a machine stopped, and one whose word is written before its mark
        const down = await component.render(<StateMark tone="bad" word="down" />);
        const stopped = await component.render(<StateMark tone="plain" word="stopped" />);
        const before = await component.render(
            <StateMark said="before" tone="new" word="creating" />,
        );

        // Then - the word is the text, read once, and no image is left to read
        expect(down.tree.text).toBe('- text: down');
        expect(stopped.tree.text).toBe('- text: stopped');
        expect(before.tree.text).toBe('- text: creating');
        expect(down.html.text).toContain('aria-hidden="true"');
    });

    test('a mark beside a sentence is hidden, the sentence says the state', async () => {
        // Given - a mark drawn beside the sentence that says it
        const result = await component.render(
            <p>
                <StateMark said="beside" tone="warn" />
                <span>It is held by its lot.</span>
            </p>,
        );

        // Then - only the sentence is read
        expect(result.tree.text).toBe('- paragraph: It is held by its lot.');
    });

    test('refuses a state with no word', async () => {
        // Given - a mark whose word is blank
        const render = component.render(<StateMark tone="ok" word=" " />);

        // Then - the render fails on the rule
        await expect(render).rejects.toThrow(
            'A state mark carries its word: a colour alone is no state.',
        );
    });

    test('the busy mark pulses while the window is visible and stands still while it is hidden (document.css)', async () => {
        // Given - a mark that pulses forever
        let states: string[] = [];
        await component.render(<StateMark tone="new" word="starting" />, async () => {
            // When - the root says visible, then hidden
            states = [playStateOf('visible'), playStateOf('hidden')];
        });
        delete document.documentElement.dataset.visibility;

        // Then - it ran, then it was paused
        expect(states).toStrictEqual(['running', 'paused']);
    });
});
