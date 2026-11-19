import { button, component } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { Switch } from '../switch';

describe('Switch', () => {
    test('its all option chooses no option, typed apart from every option', async () => {
        // Given - a switch of two machines with an all option, one machine named "all"
        const chosen: (null | string)[] = [];
        const result = await component.render(
            <Switch
                all="All machines"
                label="Which machines"
                onChange={(value) => {
                    chosen.push(value);
                }}
                options={[
                    { id: 'all', label: 'all' },
                    { id: 'mini', label: 'mini' },
                ]}
                value={null}
            />,
            async (visitor) => {
                // When - the machine named all, then the all option, are pressed
                await visitor.click(button('all'));
                await visitor.click(button('All machines'));
            },
        );

        // Then - the machine is its name, the all option is null, and all is pressed first
        expect(chosen).toStrictEqual(['all', null]);
        expect(result.tree.text).toBe(
            '- group "Which machines":\n  - button "All machines" [pressed]\n  - button "all"\n  - button "mini"',
        );
    });

    test('a disabled switch shows its choice and takes none', async () => {
        // Given - a disabled switch on apps
        const chosen: string[] = [];
        const result = await component.render(
            <Switch
                disabled
                label="Which packages"
                onChange={(value) => {
                    chosen.push(value);
                }}
                options={[
                    { id: 'apps', label: 'Apps' },
                    { id: 'settings', label: 'Settings' },
                ]}
                value="apps"
            />,
        );

        // Then - the group and each option are disabled, and nothing was chosen
        expect(result.tree.text).toBe(
            '- group "Which packages" [disabled]:\n  - button "Apps" [disabled] [pressed]\n  - button "Settings" [disabled]',
        );
        expect(chosen).toStrictEqual([]);
    });

    test('an even switch may press none, and its busy option takes no press', async () => {
        // Given - three even terms, none chosen, 4 h busy
        const chosen: string[] = [];
        const result = await component.render(
            <Switch
                even
                label="How long"
                onChange={(value) => {
                    chosen.push(value);
                }}
                options={[
                    { id: '1h', label: '1 h' },
                    { busy: true, id: '4h', label: '4 h' },
                    { id: 'until', label: 'Until 18:00' },
                ]}
                value={null}
            />,
            async (visitor) => {
                // When - Until 18:00 is pressed
                await visitor.click(button('Until 18:00'));
            },
        );

        // Then - no option is pressed, the busy one is disabled and busy, and the press reached onChange
        expect(result.tree.text).toBe(
            '- group "How long":\n  - button "1 h"\n  - button "4 h" [disabled]\n  - button "Until 18:00"',
        );
        expect(result.html.text.match(/aria-busy="true"/gu)).toHaveLength(1);
        expect(chosen).toStrictEqual(['until']);
    });

    test('a tab bar is one tab stop, the chosen tab, and each tab names the panel it controls', async () => {
        // Given - a tab bar of three tabs on Ports, controlling the panel "the-panel"
        const result = await component.render(
            <Switch
                controls="the-panel"
                label="Tabs"
                onChange={() => undefined}
                options={[
                    { id: 'Overview', label: 'Overview' },
                    { id: 'Ports', label: 'Ports' },
                    { id: 'Logs', label: 'Logs' },
                ]}
                value="Ports"
                variant="tabs"
            />,
        );

        // Then - only Ports is reached by Tab, and every tab controls the panel
        expect(result.html.text.match(/tabindex="-1"/gu)).toHaveLength(2);
        expect(result.html.text).not.toMatch(/aria-selected="true"[^>]*tabindex/u);
        expect(result.html.text.match(/aria-controls="the-panel"/gu)).toHaveLength(3);
    });
});
