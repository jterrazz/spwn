import type { Target } from '@jterrazz/station-api';

import { targetOf } from '../../model/machines';
import type { Found } from '../../model/machines';

// invariant: the one place a sign-in call names its machine. A call about a VM names it in `Machine` to this Mac's
// invariant: engine, which knows whether the VM is a far Mac's; a Mac is reached by its target, this Mac by neither.
export type Aimed<I> = { input: I; target: Target | undefined };

export function aimed<I extends object>(found: Found, input: I): Aimed<I> {
    if (found.vm === undefined) {
        return { input, target: targetOf(found) };
    }
    return { input: { ...input, Machine: found.vm.Name }, target: undefined };
}
