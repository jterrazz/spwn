import type { MachinesRelease, MachinesSSHTarget, PortsSharedPort } from '@jterrazz/station-api';

import type { Opening, Opens } from '../../application/ports/opener.port';
import { shareOpensFromHere } from '../words/provisional';

// invariant: an Openable is made from an engine answer alone, by the functions below: its constructor is
// invariant: private, so no text a page holds can reach a terminal or a browser.
export class Openable implements Opening {
    readonly opens: Opens;
    readonly value: string;

    private constructor(opens: Opens, value: string) {
        this.opens = opens;
        this.value = value;
    }

    static ofSsh(target: MachinesSSHTarget): Openable | undefined {
        return target.Command === '' ? undefined : new Openable('terminal', target.Command);
    }

    static ofRelease(release: MachinesRelease): Openable | undefined {
        return release.Notes === undefined || release.Notes === ''
            ? undefined
            : new Openable('url', release.Notes);
    }

    static ofShare(share: PortsSharedPort, onThisMac: boolean): Openable | undefined {
        const url = shareOpensFromHere(share, onThisMac);
        return share.http && url !== '' ? new Openable('url', url) : undefined;
    }
}

export function shareAddressOf(share: PortsSharedPort): string {
    return share.urls.public === '' ? share.urls.local : share.urls.public;
}
