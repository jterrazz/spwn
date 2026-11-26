import type { MachinesMacView } from '@jterrazz/station-api';

import { engineViewStopped } from '../words/shown';
import { NEW_MACHINE_WORD } from '../words/words';
import type { Route } from './route';

// value: the one button at the bar's right: Home's New machine, the Passwords page's Add a password.
export type BarAction = { label: string; onClick: () => void; leads: boolean };

// value: while this Mac's engine is stopped, Start the engine leads Home and New machine stands back.
export function newMachineLeads(macs: readonly MachinesMacView[]): boolean {
    const thisMac = macs.find((mac) => mac.ThisMac);
    return thisMac === undefined || !engineViewStopped(thisMac.Engine);
}

export type Opens = {
    route: Route;
    macs: readonly MachinesMacView[];
    newMachine: () => void;
    addPassword: () => void;
};

// value: Home's bar leads with New machine; the vault's list offers Add a password, which leads nothing.
export function actionOf({ route, macs, newMachine, addPassword }: Opens): BarAction | undefined {
    if (route.page === 'home') {
        return { label: NEW_MACHINE_WORD, leads: newMachineLeads(macs), onClick: newMachine };
    }
    if (route.page === 'passwords' && route.item === undefined) {
        return { label: 'Add a password', leads: false, onClick: addPassword };
    }
    return undefined;
}
