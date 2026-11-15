import { button, component, dialog, field, focused } from '@jterrazz/test';
import { useState } from 'react';
import { describe, expect, test, vi } from 'vitest';

import { Button } from '../button';
import { Dialog } from '../dialog';
import { FormField } from '../form-field';

function Host({ onClose }: { onClose: () => void }) {
    const [open, setOpen] = useState(false);
    const [name, setName] = useState('');
    return (
        <>
            <Button
                onClick={() => {
                    setOpen(true);
                }}
            >
                Open
            </Button>
            {open ? (
                <Dialog
                    actions={
                        <>
                            <Button>Cancel</Button>
                            <Button variant="primary">Stop</Button>
                        </>
                    }
                    onClose={() => {
                        onClose();
                        setOpen(false);
                    }}
                    title="Stop linux-1?"
                >
                    <FormField label="Name" onChange={setName} placeholder="linux-1" value={name} />
                </Dialog>
            ) : null}
        </>
    );
}

// hazard: press wraps a key longer than one letter in braces, so a chord is user-event's hold and release
// hazard: with its outer braces left to press.
const SHIFT_TAB = 'Shift>}{Tab}{/Shift';

describe('Dialog', () => {
    test('tab cycles inside the dialog, Escape closes it once, and the keyboard returns to its opener', async () => {
        // Given - a dialog opened from a button, whose onClose is a fresh closure each render
        const onClose = vi.fn<() => void>();
        await component.render(<Host onClose={onClose} />, async (visitor) => {
            await visitor.click(button('Open'));
            await visitor.see(focused(field('Name')));

            // When - Tab is pressed three times, Shift+Tab once, then a name typed and Escape pressed
            await visitor.press('Tab');
            await visitor.see(focused(button('Cancel')));
            await visitor.press('Tab');
            await visitor.see(focused(button('Stop')));
            await visitor.press('Tab');
            await visitor.see(focused(field('Name')));
            await visitor.press(SHIFT_TAB);
            await visitor.see(focused(button('Stop')));
            await visitor.fill(field('Name'), 'linux');
            await visitor.press('Escape');

            // Then - the dialog is gone and the keyboard is on Open again
            await visitor.gone(dialog('Stop linux-1?'));
            await visitor.see(focused(button('Open')));
        });

        // Then - its owner heard the dismissal exactly once
        expect(onClose).toHaveBeenCalledOnce();
    });
});
