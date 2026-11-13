import { button, component, content } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { MachineChip } from '../machine-chip';
import { Row } from '../row';

const SHORT = 'npm sign-in expired';
const LONG =
    'com.jterrazz.brain.x-snapshot is failing, last exit 2, after its third retry this hour';

function leftOf(text: string): number {
    return (
        [...document.querySelectorAll('div, span')]
            .find((one) => one.textContent === text)
            ?.getBoundingClientRect().x ?? 0
    );
}

describe('Row', () => {
    test('a row that opens keeps its link a gesture of its own, named apart', async () => {
        // Given - an account row that opens, its machine as a chip in its link slot
        const opened: string[] = [];
        const result = await component.render(
            <Row
                link={
                    <MachineChip
                        kind="workspace"
                        name="gate4"
                        onOpen={() => {
                            opened.push('machine');
                        }}
                    />
                }
                onOpen={() => {
                    opened.push('row');
                }}
                title="GitHub"
            />,
            async (visitor) => {
                // When - the chip, then the row, are clicked
                await visitor.click(button('gate4'));
                await visitor.click(button('GitHub'));
            },
        );

        // Then - each opened its own, and neither button holds the other
        expect(opened).toStrictEqual(['machine', 'row']);
        expect(result.tree.text).toBe('- button "GitHub"\n- button "gate4"');
    });

    test('a title is offered as a tooltip only when the row cuts it', async () => {
        // Given - two rows 320 px wide, one title that fits and one cut by its ellipsis
        const result = await component.render(
            <div style={{ width: 320 }}>
                <Row title={SHORT} />
                <Row title={LONG} />
            </div>,
            async (visitor) => {
                // When - the pointer rests on each title
                await visitor.hover(content(SHORT));
                await visitor.hover(content(LONG));
            },
        );

        // Then - the cut title carries its whole text as a tooltip, the one that fits none
        expect(result.html.text).toContain(`title="${LONG}"`);
        expect(result.html.text).not.toContain(`title="${SHORT}"`);
    });

    test('a nested row moves its mark and name in, and keeps its cells in their columns', async () => {
        // Given - a Mac's row and a VM's row nested under it, in the meters template
        let shift = { cell: -1, name: -1 };
        await component.render(
            <div style={{ width: 640 }}>
                <Row cells={['31 %', '6 / 64']} lead={<span />} template="meters" title="mac" />
                <Row
                    cells={['12 %', '2 / 4']}
                    lead={<span />}
                    nested
                    template="meters"
                    title="infra"
                />
            </div>,
            async (visitor) => {
                // When - both are drawn
                await visitor.see(content('infra'));
                shift = {
                    cell: leftOf('12 %') - leftOf('31 %'),
                    name: leftOf('infra') - leftOf('mac'),
                };
            },
        );

        // Then - the VM's name sits 14 px in, its first cell where the Mac's is
        expect(shift).toStrictEqual({ cell: 0, name: 14 });
    });
});
