import type { MachinesAddInput } from '@jterrazz/station-api';

export type MacDraft = { name: string; user: string; key: string };

export const EMPTY_MAC: MacDraft = { key: '', name: '', user: '' };

export function addInputOf(draft: MacDraft): MachinesAddInput {
    const name = draft.name.trim();
    const user = draft.user.trim();
    const key = draft.key.trim();
    return {
        ...(name === '' ? {} : { Name: name }),
        ...(user === '' ? {} : { User: user }),
        ...(key === '' ? {} : { Key: key }),
    };
}
