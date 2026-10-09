import type { Organization } from '../../domain/model';

export interface RoleDraft {
    id: string;
    name: string;
    level: number;
    reportsTo: string;
    canCommand: string;
    permissions: string;
}

let drafted = 0;

/** A blank role row of the creation form, with an id of its own for React. */
export function emptyRoleDraft(): RoleDraft {
    drafted += 1;
    return {
        canCommand: '',
        id: `draft-${drafted}`,
        level: 0,
        name: '',
        permissions: '',
        reportsTo: '',
    };
}

function listOf(text: string): string[] | undefined {
    const items = text
        .split(',')
        .map((item) => item.trim())
        .filter((item) => item !== '');
    return items.length === 0 ? undefined : items;
}

function slugOf(name: string): string {
    return name
        .toLowerCase()
        .replaceAll(/[^a-z0-9]+/gu, '-')
        .replaceAll(/^-|-$/gu, '');
}

/** The organization a filled form declares, or the sentence saying what it still lacks. */
export function organizationOf(
    name: string,
    description: string,
    drafts: RoleDraft[],
): Organization | string {
    const named = drafts.filter((draft) => draft.name.trim() !== '');
    if (name.trim() === '') {
        return 'Name is required';
    }
    if (named.length === 0) {
        return 'At least one role with a name is required';
    }
    return {
        description: description.trim() === '' ? undefined : description.trim(),
        name: name.trim(),
        roles: named.map((draft) => ({
            can_command: listOf(draft.canCommand),
            level: draft.level,
            name: draft.name.trim(),
            permissions: listOf(draft.permissions),
            reports_to: draft.reportsTo.trim() === '' ? undefined : draft.reportsTo.trim(),
        })),
        slug: slugOf(name.trim()),
    };
}
