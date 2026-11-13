import type {
    AppsReport,
    EngineReading,
    MachinesEngineView,
    MachinesMacView,
    MachinesVM,
    OverviewNeed,
    PortsPort,
} from '@jterrazz/station-api';

import type { Tone } from '../tone';
import {
    engineStateOf,
    engineViewStateOf,
    machineKindOf,
    macStateOf,
    shareEndOf,
    shareEndWordOf,
    vmLifetimeOf,
    vmStateOf,
} from './provisional';
import {
    APP_WHY_TONES,
    ENGINE_STATE_TONES,
    MAC_STATE_TONES,
    NEWEST_TONE,
    OWN_STATE_TONES,
    PORT_REACH_TONES,
    REVERSIBLE_TONE,
    SEVERITY_TONES,
    SHARE_END_TONES,
    VM_CHECK_TONES,
    VM_STATE_TONES,
} from './tones';
import {
    APP_WHY_WORDS,
    ENGINE_STATE_WORDS,
    LIFETIME_WORDS,
    MAC_STATE_WORDS,
    MACHINE_KIND_WORDS,
    NEWEST_WORD,
    newestOutWord,
    NO_STATE_WORD,
    OWN_STATE_WORDS,
    PORT_REACH_WORDS,
    REVERSIBLE_WORD,
    SEVERITY_WORDS,
    SHARE_TO_WORDS,
    SSH_WORDS,
    SYSTEM_WORDS,
    VM_CHECK_WORDS,
    VM_STATE_WORDS,
} from './words';
import type { OwnState } from './words';

export type Shown = { tone: Tone; word: string };

export function ownState(state: OwnState): Shown {
    return { tone: OWN_STATE_TONES[state], word: OWN_STATE_WORDS[state] };
}

export function vmState(vm: MachinesVM): Shown {
    const state = vmStateOf(vm);
    if (state === undefined) {
        return { tone: 'plain', word: vm.State === '' ? NO_STATE_WORD : vm.State };
    }
    return { tone: VM_STATE_TONES[state], word: VM_STATE_WORDS[state] };
}

export function vmCheck(vm: MachinesVM): Shown {
    const state = vm.CheckStatus.state ?? '';
    return { tone: VM_CHECK_TONES[state], word: VM_CHECK_WORDS[state] };
}

export function vmSsh(vm: MachinesVM): Shown {
    return { tone: 'plain', word: vm.SSH ? SSH_WORDS.answers : SSH_WORDS.silent };
}

export function macState(mac: MachinesMacView): Shown {
    const state = macStateOf(mac);
    return { tone: MAC_STATE_TONES[state], word: MAC_STATE_WORDS[state] };
}

export function needState(need: OverviewNeed): Shown {
    return {
        tone: SEVERITY_TONES[need.Severity],
        word: SEVERITY_WORDS[need.Severity],
    };
}

export function engineState(reading: EngineReading): Shown {
    const state = engineStateOf(reading);
    if (state === undefined) {
        return { tone: 'plain', word: reading.State === '' ? NO_STATE_WORD : reading.State };
    }
    return { tone: ENGINE_STATE_TONES[state], word: ENGINE_STATE_WORDS[state] };
}

export function portReach(port: PortsPort): Shown {
    return {
        tone: PORT_REACH_TONES[port.Reach] ?? 'plain',
        word: PORT_REACH_WORDS[port.Reach] ?? `reach ${port.Reach}, not worded`,
    };
}

export function appUpdate(report: AppsReport): null | Shown {
    if (report.Why !== 'outdated') {
        return null;
    }
    return {
        tone: APP_WHY_TONES.outdated,
        word: report.Newest === '' ? APP_WHY_WORDS.outdated : newestOutWord(report.Newest),
    };
}

export function machineKindWord(kind: string): string {
    return MACHINE_KIND_WORDS[kind] ?? kind;
}

export function newestState(newest: boolean): null | Shown {
    return newest ? { tone: NEWEST_TONE, word: NEWEST_WORD } : null;
}

export function reversibleState(reversible: boolean): null | Shown {
    return reversible ? { tone: REVERSIBLE_TONE, word: REVERSIBLE_WORD } : null;
}

export function holdsFor(until: string, now: string): Shown {
    const end = shareEndOf(until, now);
    return { tone: SHARE_END_TONES[end.kind], word: shareEndWordOf(end, until) };
}

export function vmAbout(vm: MachinesVM): { lifetime: string; system: string } {
    const lifetime = vmLifetimeOf(vm);
    return {
        lifetime: lifetime === undefined ? vm.Lifetime : LIFETIME_WORDS[lifetime],
        system: SYSTEM_WORDS[vm.Kind],
    };
}

export function vmKindWord(vm: Pick<MachinesVM, 'Lifetime'>): string {
    return machineKindWord(machineKindOf(vm));
}

export function engineViewWord(engine: MachinesEngineView): string | undefined {
    const state = engineViewStateOf(engine);
    return state === undefined ? undefined : ENGINE_STATE_WORDS[state];
}

function isCodeOf<T extends PropertyKey>(
    words: Readonly<Record<T, string>>,
    code: PropertyKey,
): code is T {
    return Object.hasOwn(words, code);
}

export function shareToWord(to: string): string {
    return isCodeOf(SHARE_TO_WORDS, to) ? SHARE_TO_WORDS[to] : to;
}
