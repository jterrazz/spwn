import type { CallName, Calls } from '@jterrazz/station-api';

// outside: packages/api carries a call's confirmation as a type only (Calls[N]['confirm']), and the window needs to
// outside: know at run time whether a call asks one before it sends; the word the engine checks is the subject
// outside: typed or the yes, never this table's. This is the set of those calls as a value, checked both ways by the
// outside: compiler: a call that gains or loses a confirmation fails the build here. It goes when packages/api
// outside: ships calls.json's confirmations as a value.

export type ConfirmName = {
    [N in CallName]: Calls[N] extends { confirm: string } ? N : never;
}[CallName];

export const CONFIRMED: Readonly<Record<ConfirmName, true>> = {
    'accounts.take-back': true,
    'agents.stop': true,
    'apps.remove': true,
    'background.remove': true,
    'disk.erase': true,
    'engine.stop': true,
    'images.delete': true,
    'machinefile.apply': true,
    'machines.cleanup': true,
    'machines.create-server': true,
    'machines.create-workspace': true,
    'machines.delete': true,
    'machines.forget': true,
    'machines.restart': true,
    'machines.restart-mac': true,
    'machines.stop': true,
    'ports.stop-sharing': true,
    'ports.undeclare': true,
    'settings.apply': true,
    'settings.off': true,
    'settings.run-containers': true,
};

export function isConfirmed(name: CallName): name is ConfirmName {
    return Object.hasOwn(CONFIRMED, name);
}
