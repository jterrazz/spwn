import { useWatch } from '../../engine/react';

// invariant: a password is named by its engine label, the id its route carries standing in until vault.list answers.
export function PasswordName({ id }: { id: string }) {
    const listed = useWatch('vault.list', {}, undefined);
    const entries = listed.kind === 'read' ? (listed.out.Entries ?? []) : [];
    return entries.find((entry) => entry.ID === id)?.Label ?? id;
}
