import type { CallName } from '@jterrazz/station-api';
import confirms from '@jterrazz/station-api/confirms.json';
import type { ConfirmKind } from '@jterrazz/station-api/confirms.json';

import type { AskForm } from './gesture';

export type { ConfirmKind };

export type Confirming = { form: AskForm | undefined; consequence: string };

const FORMS = { none: undefined, 'type-name': 'name', 'yes-no': 'yes' } as const;

export function confirmingOf(call: CallName, kind?: ConfirmKind): Confirming | undefined {
    const base = confirms[call];
    if (base === undefined) {
        return undefined;
    }
    const { form, consequence } = (kind === undefined ? undefined : base.kinds?.[kind]) ?? base;
    return { consequence, form: FORMS[form] };
}

export function consequenceOf(
    confirming: Confirming | undefined,
    subject: string,
): string | undefined {
    return confirming?.consequence.replace('%s', () => subject);
}
