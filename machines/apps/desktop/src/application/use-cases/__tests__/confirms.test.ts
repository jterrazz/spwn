import { describe, expect, test } from 'vitest';

import { confirmingOf, consequenceOf } from '../confirms';

describe('confirms', () => {
    test("a call's confirmation is the engine's, its kind of machine choosing among the forms it gives", () => {
        // Given - machines.restart, which a server types, a workspace skips and a Mac answers yes to, and machines.stop

        // When - each is asked for its kind
        const forms = [
            confirmingOf('machines.restart', 'server')?.form,
            confirmingOf('machines.restart', 'workspace')?.form,
            confirmingOf('machines.restart', 'computer')?.form,
            confirmingOf('machines.restart')?.form,
            confirmingOf('machines.stop', 'server')?.form,
            confirmingOf('machines.stop', 'workspace')?.form,
            confirmingOf('machines.delete', 'workspace')?.form,
        ];

        // Then - the name, nothing, a yes, a yes, the name, a yes, the name
        expect(forms).toStrictEqual(['name', undefined, 'yes', 'yes', 'name', 'yes', 'name']);
    });

    test('a call that asks nothing has no confirmation', () => {
        // Given - machines.start

        // Then - none
        expect(confirmingOf('machines.start')).toBeUndefined();
    });

    test("a consequence is the engine's sentence, its subject put where the engine marks it", () => {
        // Given - machines.stop, which names its subject, engine.stop, which names none, and a call that asks nothing

        // When - each consequence is asked
        const said = [
            consequenceOf(confirmingOf('machines.stop'), 'linux-1'),
            consequenceOf(confirmingOf('engine.stop'), 'linux-1'),
            consequenceOf(confirmingOf('machines.start'), 'linux-1'),
        ];

        // Then - the sentence with the subject filled, the sentence as it is, and nothing
        expect(said).toStrictEqual([
            'linux-1 stops: what it serves stops until you start it again.',
            'Every VM of this Mac stops, and nothing is shared until the engine starts again.',
            undefined,
        ]);
    });
});
