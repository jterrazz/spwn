import { button, component, content } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { MachineChip } from '../machine-chip';
import { Row } from '../row';

const SHORT = 'npm sign-in expired';
const LONG =
    'com.jterrazz.brain.x-snapshot is failing, last exit 2, after its third retry this hour';

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
});
