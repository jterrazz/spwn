import type { CallName } from '@jterrazz/station-api';
import confirms from '@jterrazz/station-api/confirms.json';
import type { ConfirmForm, ConfirmKind } from '@jterrazz/station-api/confirms.json';

import type { AskForm } from './gesture';

export type { ConfirmForm, ConfirmKind };

export type Confirming = { form: AskForm | undefined; consequence: string };

const FORMS = { none: undefined, 'type-name': 'name', 'yes-no': 'yes' } as const;

// invariant: `asks` is the form the engine graded for this very target (a VM's `Asks`), which wins over the call's
// invariant: table by kind: both are the engine's one grading (core/confirm), the first read with the target's facts.
export function confirmingOf(
    call: CallName,
    kind?: ConfirmKind,
    asks?: '' | ConfirmForm,
): Confirming | undefined {
    const base = confirms[call];
    if (base === undefined) {
        return undefined;
    }
    const { form, consequence } = (kind === undefined ? undefined : base.kinds?.[kind]) ?? base;
    return { consequence, form: FORMS[asks === undefined || asks === '' ? form : asks] };
}

export function consequenceOf(
    confirming: Confirming | undefined,
    subject: string,
): string | undefined {
    return confirming?.consequence.replace('%s', () => subject);
}
