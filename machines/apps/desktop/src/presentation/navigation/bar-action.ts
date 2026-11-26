import type { MachinesMacView, OverviewNeed } from '@jterrazz/station-api';

import { engineViewStopped, leadingNeed } from '../words/shown';
import { NEW_MACHINE_WORD } from '../words/words';
import type { Route } from './route';

// value: the one button at the bar's right: Home's New machine, the Passwords page's Add a password.
export type BarAction = { label: string; onClick: () => void; leads: boolean };

// value: New machine leads Home only while nothing is broken for the owner to fix: while this Mac's engine is
// value: stopped, Start the engine leads, and while a need is broken, its gesture does, the one white button of the page.
export function newMachineLeads(
    macs: readonly MachinesMacView[],
    needs: readonly OverviewNeed[],
): boolean {
    const thisMac = macs.find((mac) => mac.ThisMac);
    return (
        (thisMac === undefined || !engineViewStopped(thisMac.Engine)) && leadingNeed(needs) === -1
    );
}

export type Opens = {
    route: Route;
    macs: readonly MachinesMacView[];
    needs: readonly OverviewNeed[];
    newMachine: () => void;
    addPassword: () => void;
};

// value: Home's bar leads with New machine; the vault's list offers Add a password, which leads nothing.
export function actionOf({
    route,
    macs,
    needs,
    newMachine,
    addPassword,
}: Opens): BarAction | undefined {
    if (route.page === 'home') {
        return {
            label: NEW_MACHINE_WORD,
            leads: newMachineLeads(macs, needs),
            onClick: newMachine,
        };
    }
    if (route.page === 'passwords' && route.item === undefined) {
        return { label: 'Add a password', leads: false, onClick: addPassword };
    }
    return undefined;
}
