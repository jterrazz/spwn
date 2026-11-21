import { describe, expect, test } from 'vitest';

import { workspaceNameError } from '../workspace-name';

describe('workspace-name', () => {
    test.each(['default', 'w1', 'my-repo', 'a', '  docs  ', ''])(
        'the name %j is accepted',
        (name) => {
            // Given - a name typed in the field

            // When - the field is checked
            const message = workspaceNameError(name);

            // Then - nothing is shown
            expect(message).toBe('');
        },
    );

    test.each(['My Repo', '1repo', '-repo', 'repo_1', 'repo.git', 'dépôt'])(
        'the name %j is refused at the field with the rule',
        (name) => {
            // Given - a name typed in the field

            // When - the field is checked
            const message = workspaceNameError(name);

            // Then - the rule is stated
            expect(message).toBe(
                'Use lowercase letters, digits and hyphens, starting with a letter',
            );
        },
    );
});
