import { identityOf } from '../../design/index';
import type { Identity } from '../../design/index';
import { machineNamesOf, macsOf } from '../../domain/machines/machines';
import { useWatch } from '../../engine/react';

export type IdentityOf = (name: string) => Identity | undefined;

export function useIdentityOf(): IdentityOf {
    const listing = useWatch('machines.list', {});
    const listed = machineNamesOf(macsOf(listing.kind === 'read' ? listing.out : undefined));
    return (name) => identityOf(name, listed);
}
