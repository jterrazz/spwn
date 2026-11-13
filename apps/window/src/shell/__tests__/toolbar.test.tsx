import { button, component } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { Button, Screen } from '../../design/index';
import { Toolbar } from '../toolbar';

const IDLE = () => undefined;

function toolbar(onNewMachine: (() => void) | undefined) {
    return (
        <Toolbar
            canGoBack={false}
            crumbs={[{ label: 'Storage' }]}
            onBack={IDLE}
            onNewMachine={onNewMachine}
            onToggleSidebar={IDLE}
        />
    );
}

describe('Toolbar', () => {
    test("a toolbar with no New machine leaves the screen's primary to its page", async () => {
        // Given - a screen holding a toolbar off Home and a page's one primary
        const result = await component.render(
            <Screen>
                {toolbar(undefined)}
                <Button variant="primary">Free what can be rebuilt</Button>
            </Screen>,
            async (visitor) => {
                // When - it is drawn
                await visitor.see(button('Free what can be rebuilt'));
            },
        );

        // Then - nothing is refused, and the toolbar draws no New machine
        expect(result.errors.text).toBe('');
        expect(result.content.text).not.toContain('New machine');
    });

    test("the New machine of Home's toolbar claims the screen's one primary", async () => {
        // Given - a screen asked to draw Home's toolbar and a page primary beside it
        const render = component.render(
            <Screen>
                {toolbar(IDLE)}
                <Button variant="primary">Free what can be rebuilt</Button>
            </Screen>,
        );

        // Then - the render fails on the rule
        await expect(render).rejects.toThrow(
            'One primary button per screen: a second one was drawn.',
        );
    });
});
