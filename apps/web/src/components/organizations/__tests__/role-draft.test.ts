import { describe, expect, test } from 'vitest';

import { emptyRoleDraft, organizationOf } from '../role-draft';

describe('role-draft', () => {
    test('a filled form declares the organization, its slug, and its roles with lists split on commas', () => {
        // Given - a named form with one filled role and one left blank
        const organization = organizationOf('  Military Chain! ', '', [
            {
                ...emptyRoleDraft(),
                canCommand: 'captain, , private',
                level: 0,
                name: ' general ',
                permissions: 'delegate',
            },
            emptyRoleDraft(),
        ]);

        // Then - the blank role is dropped, empty text becomes absent
        expect(organization).toStrictEqual({
            description: undefined,
            name: 'Military Chain!',
            roles: [
                {
                    can_command: ['captain', 'private'],
                    level: 0,
                    name: 'general',
                    permissions: ['delegate'],
                    reports_to: undefined,
                },
            ],
            slug: 'military-chain',
        });
    });

    test('a form missing its name, or every role name, says what it lacks', () => {
        // Given - a blank name, then a name with only blank roles
        // Then - the name is asked for first, then the roles
        expect(organizationOf(' ', '', [{ ...emptyRoleDraft(), name: 'chief' }])).toBe(
            'Name is required',
        );
        expect(organizationOf('Crew', '', [emptyRoleDraft()])).toBe(
            'At least one role with a name is required',
        );
    });
});
