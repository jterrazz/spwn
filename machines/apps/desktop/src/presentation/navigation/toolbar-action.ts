import type { MachinesMacView } from '@jterrazz/station-api';

import { engineViewStopped } from '../words/shown';
import { NEW_MACHINE_WORD } from '../words/words';
import type { Route } from './route';
import type { ToolbarAction } from './toolbar';

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

// value: Home's toolbar leads with New machine; the vault's list offers Add a password, which leads nothing.
export function actionOf({
    route,
    macs,
    newMachine,
    addPassword,
}: Opens): ToolbarAction | undefined {
    if (route.page === 'home') {
        return { label: NEW_MACHINE_WORD, leads: newMachineLeads(macs), onClick: newMachine };
    }
    if (route.page === 'passwords' && route.item === undefined) {
        return { label: 'Add a password', leads: false, onClick: addPassword };
    }
    return undefined;
}
