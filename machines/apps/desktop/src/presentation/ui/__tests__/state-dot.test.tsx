import { component } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import '../document.css';
import { StateDot } from '../state-dot';

function playStateOf(visibility: 'hidden' | 'visible'): string {
    document.documentElement.dataset.visibility = visibility;
    const dot = document.querySelector('[role="img"]');
    return dot === null ? '' : getComputedStyle(dot).animationPlayState;
}

describe('StateDot', () => {
    test('a dot that pulses runs while the window is visible and stands still while it is hidden (document.css)', async () => {
        // Given - a dot that pulses forever
        let states: string[] = [];
        await component.render(<StateDot tone="new" word="Starting" />, async () => {
            // When - the root says visible, then hidden
            states = [playStateOf('visible'), playStateOf('hidden')];
        });
        delete document.documentElement.dataset.visibility;

        // Then - it ran, then it was paused
        expect(states).toStrictEqual(['running', 'paused']);
    });
});
