import { describe, expect, test } from 'vitest';

import { emptyRoleDraft, organizationOf } from '../role-draft';

describe('role-draft', () => {
    test('a new role row is blank at level 0, under the id it is given', () => {
        // Given - the row keyed draft-2

        // Then - every field empty
        expect(emptyRoleDraft('draft-2')).toStrictEqual({
            canCommand: '',
            id: 'draft-2',
            level: 0,
            name: '',
            permissions: '',
            reportsTo: '',
        });
    });

    test('a filled form declares the organization, its slug, and its roles with lists split on commas', () => {
        // Given - a named form, a role with its lists, one with only a superior, one with a blank name
        const organization = organizationOf('  Military — Chain! ', '  For war ', [
            {
                ...emptyRoleDraft('general'),
                canCommand: 'captain, , private',
                name: ' general ',
                permissions: 'delegate',
                reportsTo: '  ',
            },
            { ...emptyRoleDraft('captain'), level: 1, name: 'captain', reportsTo: ' general ' },
            { ...emptyRoleDraft('blank'), name: '   ' },
        ]);

        // Then - the blank role is dropped, every text trimmed, empty text absent
        expect(organization).toStrictEqual({
            description: 'For war',
            name: 'Military — Chain!',
            roles: [
                {
                    can_command: ['captain', 'private'],
                    level: 0,
                    name: 'general',
                    permissions: ['delegate'],
                    reports_to: undefined,
                },
                {
                    can_command: undefined,
                    level: 1,
                    name: 'captain',
                    permissions: undefined,
                    reports_to: 'general',
                },
            ],
            slug: 'military-chain',
        });
    });

    test('a blank description is absent', () => {
        // Given - a crew described by spaces alone
        const organization = organizationOf('Crew', '   ', [
            { ...emptyRoleDraft('chief'), name: 'chief' },
        ]);

        // Then - no description
        expect(organization).toStrictEqual({
            description: undefined,
            name: 'Crew',
            roles: [
                {
                    can_command: undefined,
                    level: 0,
                    name: 'chief',
                    permissions: undefined,
                    reports_to: undefined,
                },
            ],
            slug: 'crew',
        });
    });

    test('a form missing its name, or every role name, says what it lacks', () => {
        // Given - a blank name, then a name with only blank roles
        // Then - the name is asked for first, then the roles
        expect(organizationOf(' ', '', [{ ...emptyRoleDraft('chief'), name: 'chief' }])).toBe(
            'Name is required',
        );
        expect(organizationOf('Crew', '', [emptyRoleDraft('blank')])).toBe(
            'At least one role with a name is required',
        );
    });
});
