import { button, component, dialog, disabled, enabled, field } from '@jterrazz/test';
import { describe, expect, test, vi } from 'vitest';

import { ConfirmDialog } from '../confirm-dialog';

function ignored(): void {
    return undefined;
}

describe('ConfirmDialog', () => {
    test('a destroying gesture confirms on the filled danger button', async () => {
        // Given - the confirm of an erase
        const result = await component.render(
            <ConfirmDialog
                destroys
                onCancel={ignored}
                onConfirm={ignored}
                title="Erase orbstack?"
                verb="Erase"
            />,
            async (visitor) => {
                await visitor.see(dialog('Erase orbstack?'));
            },
        );

        // Then - its verb is drawn filled in danger, and nothing is drawn primary
        expect(result.html.text).toMatch(/class="[^"]*destroy[^"]*"[^>]*>Erase</u);
        expect(result.html.text).not.toMatch(/class="[^"]*primary/u);
    });

    test('a gesture that destroys nothing confirms on the primary', async () => {
        // Given - the confirm of a stop
        const result = await component.render(
            <ConfirmDialog
                onCancel={ignored}
                onConfirm={ignored}
                title="Stop linux-1?"
                verb="Stop"
            />,
            async (visitor) => {
                await visitor.see(button('Stop'));
            },
        );

        // Then - its verb is the primary, and nothing is drawn in danger
        expect(result.html.text).toMatch(/class="[^"]*primary[^"]*"[^>]*>Stop</u);
        expect(result.html.text).not.toMatch(/class="[^"]*destroy/u);
    });

    test('a Return in the field confirms once the name typed matches', async () => {
        // Given - the confirm of an erase asking orbstack typed
        const onConfirm = vi.fn<(typed: string) => void>();
        await component.render(
            <ConfirmDialog
                onCancel={ignored}
                onConfirm={onConfirm}
                title="Erase orbstack?"
                typed="orbstack"
                verb="Erase"
            />,
            async (visitor) => {
                // When - the name is typed, then Return pressed in the field
                await visitor.fill(field('Type orbstack to confirm'), 'orbstack');
                await visitor.see(enabled(button('Erase')));
                await visitor.press('Enter');
            },
        );

        // Then - the erase is confirmed with the name, once
        expect(onConfirm).toHaveBeenCalledExactlyOnceWith('orbstack');
    });

    test('a Return in the field does nothing while the confirm button is disabled', async () => {
        // Given - the confirm of an erase asking orbstack typed
        const onConfirm = vi.fn<(typed: string) => void>();
        await component.render(
            <ConfirmDialog
                onCancel={ignored}
                onConfirm={onConfirm}
                title="Erase orbstack?"
                typed="orbstack"
                verb="Erase"
            />,
            async (visitor) => {
                // When - a name that differs is typed, then Return pressed in the field
                await visitor.fill(field('Type orbstack to confirm'), 'orbstac');
                await visitor.see(disabled(button('Erase')));
                await visitor.press('Enter');
                await visitor.see(dialog('Erase orbstack?'));
            },
        );

        // Then - nothing is confirmed
        expect(onConfirm).not.toHaveBeenCalled();
    });

    test('the field asking the name opens empty, what to type said faint inside it', async () => {
        // Given - the confirm of a delete asking api typed
        const result = await component.render(
            <ConfirmDialog
                onCancel={ignored}
                onConfirm={ignored}
                title="Delete api?"
                typed="api"
                verb="Delete api"
            />,
            async (visitor) => {
                await visitor.see(field('Type api to confirm'));
            },
        );

        // Then - what to type is said inside the field, faint, and the field holds no text of its own
        expect(result.html.text).toMatch(/placeholder="Type api to confirm"/u);
        expect(result.html.text).toMatch(/value=""/u);
        expect(result.html.text).not.toMatch(/<label/u);
    });

    test("a hold the engine names is drawn after the gesture's consequence", async () => {
        // Given - the confirm of a delete of a held machine
        const result = await component.render(
            <ConfirmDialog
                held="It is held by its agent: work may be in progress."
                onCancel={ignored}
                onConfirm={ignored}
                said="linux-1 and everything on its disk are deleted for good."
                title="Delete linux-1?"
                verb="Delete"
            />,
            async (visitor) => {
                await visitor.see(dialog('Delete linux-1?'));
            },
        );

        // Then - both sentences are drawn, the hold last
        expect(result.html.text).toMatch(
            /linux-1 and everything on its disk are deleted for good\. It is held by its agent: work may be in progress\./u,
        );
    });
});
