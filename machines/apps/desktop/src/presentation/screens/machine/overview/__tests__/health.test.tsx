import { button, clock, component, file, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../../../composition/__tests__/page-over';
import { recordingsOf } from '../../../../../infrastructure/outbound/engine-replay/recordings';
import { ageOf } from '../health';

const recordingFor = await recordingsOf('working-hard');

const RESTARTED = 'Sun 18 Oct. Down from 04:30 to 04:34, out of memory; restarted 1 time';

function infraOverview() {
    return pageOver(recordingFor('working-hard'), {
        machine: 'infra',
        page: 'machine',
        tab: 'Overview',
    });
}

describe('health', () => {
    test('a check is aged in seconds, minutes or hours within a day, in days past it', () => {
        // Given - the window read on 13 Nov at 10:00
        using _ = clock.at('2026-11-13T10:00:00Z');

        // Then - each check's age from then, and none for a check never run
        expect(ageOf('2026-11-13T09:59:57Z')).toBe('3 s ago');
        expect(ageOf('2026-11-13T07:00:00Z')).toBe('3 h ago');
        expect(ageOf('2026-11-07T02:00:00Z')).toBe('6 d 8 h ago');
        expect(ageOf(undefined)).toBeUndefined();
    });

    test("a server's last 30 days are the engine's strip, each day named by its sentence", async () => {
        // Given - infra's Overview over working-hard, a server restarted on two of its 30 days
        using _ = clock.at('2026-11-07T10:00:00Z');
        const result = await component.render(infraOverview(), async (visitor) => {
            // When - its strip has been drawn from machines.list
            const days = region('Last 30 days');
            await visitor.see(within(days, button(RESTARTED)));
        });

        // Then - the strip's sentence heads the panel, a restarted day is a button, a calm one a named cell
        expect(result.tree.text).toContain('Up 99.98 % over 30 days, 2 restarts');
        await expect(result.tree).toMatch(
            file('./__expected__/working-hard-infra-strip.aria.yaml'),
        );
    });

    test('a restarted day of the strip opens the Inbox on its history', async () => {
        // Given - infra's Overview over working-hard
        const result = await component.render(infraOverview(), async (visitor) => {
            const restarted = within(region('Last 30 days'), button(RESTARTED));
            await visitor.see(restarted);

            // When - the restarted day is pressed
            await visitor.click(restarted);

            // Then - the Inbox shows its History
            await visitor.see(region('History'));
        });
        expect(result.errors.text).toBe('');
    });
});
