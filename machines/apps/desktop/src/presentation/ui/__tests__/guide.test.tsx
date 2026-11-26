import { button, component, dialog, file, focused, status } from '@jterrazz/test';
import { useState } from 'react';
import { describe, expect, test, vi } from 'vitest';

import { Button } from '../button';
import { Guide } from '../guide';
import type { GuideProps, GuideView } from '../guide';
import { GUIDE_STORIES } from '../guide.stories';

const GOLDENS = [
    './__expected__/guide-working-measured.aria.yaml',
    './__expected__/guide-working-live.aria.yaml',
    './__expected__/guide-waiting-here.aria.yaml',
    './__expected__/guide-waiting-elsewhere.aria.yaml',
    './__expected__/guide-failed.aria.yaml',
    './__expected__/guide-done.aria.yaml',
    './__expected__/guide-you-may-leave.aria.yaml',
    './__expected__/guide-two-steps.aria.yaml',
    './__expected__/guide-seven-steps.aria.yaml',
    './__expected__/guide-folded.aria.yaml',
] as const;

const DRAWN = GOLDENS.map((golden, at) => {
    const story = GUIDE_STORIES[at];
    if (story === undefined) {
        throw new Error(`no story for ${golden}`);
    }
    return { golden, name: story.name, props: story.props };
});

function storyOf(name: string): GuideProps {
    const story = GUIDE_STORIES.find((one) => one.name === name);
    if (story === undefined) {
        throw new Error(`no story named ${name}`);
    }
    return story.props;
}

const HERE = storyOf('waiting on you, here');

const SIGNING: GuideView = {
    headline: { next: 'Touch the sensor on this Mac.', now: 'Waiting for your fingerprint.' },
    illustration: 'touch-id',
    move: 'owner',
    progress: { kind: 'live', line: 'macOS prompt open · 4 s' },
    sentence: 'macOS is asking now. Use your password if Touch ID is off.',
};

type Heard = { onHide?: () => void; onSign?: () => void };

function Host({ onHide, onSign }: Heard) {
    const [open, setOpen] = useState(false);
    const [view, setView] = useState<GuideView>(HERE.view);
    const signing = {
        ...view,
        action:
            view.action === undefined
                ? undefined
                : {
                      ...view.action,
                      onPress: () => {
                          onSign?.();
                          setView(SIGNING);
                      },
                  },
    };
    return (
        <>
            <Button
                onClick={() => {
                    setOpen(true);
                }}
            >
                Add a Mac
            </Button>
            {open ? (
                <Guide
                    {...HERE}
                    onHide={
                        onHide === undefined
                            ? undefined
                            : () => {
                                  onHide();
                                  setOpen(false);
                              }
                    }
                    view={signing}
                />
            ) : null}
        </>
    );
}

describe('Guide', () => {
    test.each(DRAWN)('$name: a dialog of named steps over the current one', async (story) => {
        // Given - the story's guide
        const result = await component.render(<Guide {...story.props} />, async (visitor) => {
            // When - it has drawn
            await visitor.see(dialog(`${story.props.act} ${story.props.subject}`));
        });

        // Then - its accessibility tree is the story's, and nothing failed
        await expect(result.tree).toMatch(file(story.golden));
        expect(result.errors.text).toBe('');
    });

    test('it opens on itself, holds Tab inside, Escape hides it once and the keyboard returns to its opener', async () => {
        // Given - a guide opened from a button, its act able to go on hidden
        const onHide = vi.fn<() => void>();
        await component.render(<Host onHide={onHide} />, async (visitor) => {
            await visitor.click(button('Add a Mac'));
            await visitor.see(focused(dialog('Add a Mac mac-mini')));

            // When - Tab walks its three controls and wraps, then Escape is pressed
            await visitor.press('Tab');
            await visitor.see(focused(button('Hide')));
            await visitor.press('Tab');
            await visitor.see(focused(button('Sign with Touch ID →')));
            await visitor.press('Tab');
            await visitor.see(focused(button('Not now')));
            await visitor.press('Tab');
            await visitor.see(focused(button('Hide')));
            await visitor.press('Escape');

            // Then - it is gone and the keyboard is on its opener again
            await visitor.gone(dialog('Add a Mac mac-mini'));
            await visitor.see(focused(button('Add a Mac')));
        });

        // Then - its owner heard Hide exactly once
        expect(onHide).toHaveBeenCalledOnce();
    });

    test('an act that cannot go on hidden draws no Hide, and Escape leaves it in place', async () => {
        // Given - a finished act, which has nothing to hide
        const done = storyOf('done');
        const result = await component.render(<Guide {...done} />, async (visitor) => {
            await visitor.see(focused(dialog('New machine linux-5')));

            // When - Escape is pressed
            await visitor.press('Escape');

            // Then - it is still there, its keyboard on it
            await visitor.see(focused(dialog('New machine linux-5')));
        });

        // Then - no Hide was drawn
        expect(result.html.text).not.toContain('>Hide<');
    });

    test('a new view is said once to assistive tech, and the keyboard stays in the guide when its button goes', async () => {
        // Given - a guide waiting on Touch ID, its keyboard on the owner's action
        const onSign = vi.fn<() => void>();
        await component.render(
            <Host onHide={vi.fn<() => void>()} onSign={onSign} />,
            async (visitor) => {
                await visitor.click(button('Add a Mac'));
                await visitor.see(status('Touch ID is next. Sign once to reach mac-mini.'));
                await visitor.press('Tab');
                await visitor.press('Tab');
                await visitor.see(focused(button('Sign with Touch ID →')));

                // When - the owner presses it, and the engine's next view has no action
                await visitor.press('Enter');

                // Then - the live region says the new view, and the guide holds the keyboard
                await visitor.see(
                    status('Waiting for your fingerprint. Touch the sensor on this Mac.'),
                );
                await visitor.gone(button('Sign with Touch ID →'));
                await visitor.see(focused(dialog('Add a Mac mac-mini')));
            },
        );

        // Then - the press was heard once
        expect(onSign).toHaveBeenCalledOnce();
    });

    test('a bar is drawn only for a measure, with its figure as its value, and never for a live line', async () => {
        // Given - a step the engine measures and one it only tells
        const measured = await component.render(<Guide {...storyOf('working, measured')} />);
        const told = await component.render(<Guide {...storyOf('working, its last line')} />);

        // Then - the measured one has one progress bar saying its figure, the told one none
        expect(measured.html.text.match(/role="progressbar"/gu)).toHaveLength(1);
        expect(measured.html.text).toContain('aria-valuetext="214 of 380 MB · about 40 s"');
        expect(told.html.text).not.toContain('role="progressbar"');
        expect(told.html.text).toContain('asked for an address 9 s ago');
    });

    test('the action is primary only when the next move is the owner’s', async () => {
        // Given - one action, on the owner's move, then on the engine's
        const here = storyOf('waiting on you, here');
        const yours = await component.render(<Guide {...here} />);
        const engines = await component.render(
            <Guide {...here} view={{ ...here.view, move: 'engine' }} />,
        );

        // Then - it is primary on the owner's move alone
        expect(yours.html.text.match(/class="[^"]*primary/gu)).toHaveLength(1);
        expect(engines.html.text.match(/class="[^"]*primary/gu)).toBeNull();
    });
});
