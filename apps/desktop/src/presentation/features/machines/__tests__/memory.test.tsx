import { component, content, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { composeReplay } from '../../../../composition/compose';
import { recordingsOf } from '../../../../infrastructure/outbound/engine-replay/recordings';
import { EngineProvider } from '../../../dependencies/dependencies';
import { MemoryFreeTile, MemoryPanel } from '../memory';

const recordingFor = await recordingsOf('creating', 'one-mac', 'two-macs');

function over(world: string, only: string) {
    const dependencies = composeReplay({ recording: recordingFor(world) });
    return (
        <EngineProvider dependencies={dependencies}>
            <MemoryFreeTile />
            <MemoryPanel macs={[]} only={only} />
        </EngineProvider>
    );
}

describe('memory', () => {
    test("the free memory is each Mac's room summed, and a Mac's grid draws what machines.list measures of it alone", async () => {
        // Given - two-macs: 4 of 16 GB free on the MacBook Pro, 36 of 64 on the mini
        const result = await component.render(over('two-macs', 'mac-mini'), async (visitor) => {
            // When - machines.list has settled with both Macs
            await visitor.see(within(region('Memory'), content('mac-mini')));
        });

        // Then - the tile sums both Macs, and the mini's grid draws macOS, its machines and its free
        expect(result.content.text).toMatch(/Memory free\s*40\s*GB[\s\S]*of 80/u);
        expect(result.content.text).toMatch(/\b36 GB free\s*of 64/u);
        expect(result.content.text).not.toMatch(/\b4 GB free\s*of 16/u);
        expect(result.content.text).toMatch(/24\s*infra/u);
        expect(result.content.text).not.toContain('room for');
    });

    test("a Mac Overview's grid is its Mac's alone, and a Mac whose room the engine does not give says so", async () => {
        // Given - the MacBook Pro over creating, then one-mac, whose Mac has no measured memory
        const creating = await component.render(
            over('creating', 'macbook-pro'),
            async (visitor) => {
                await visitor.see(content('linux-4', { exact: false }));
            },
        );
        const silent = await component.render(over('one-mac', 'macbook-pro'), async (visitor) => {
            // When - machines.list has answered with no room
            await visitor.see(content("What each machine holds of this Mac's memory."));
        });

        // Then - the machine being created holds its squares, and the silent Mac draws no number
        expect(creating.content.text).toMatch(/4\s*linux-5/u);
        expect(silent.content.text).not.toContain('GB free');
    });
});
