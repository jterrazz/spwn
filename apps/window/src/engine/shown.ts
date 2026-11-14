import type {
    AppsReport,
    CoreVerdictVerdict,
    EngineReading,
    MachinefileChapter,
    MachinefileDiffering,
    MachinesBirth,
    MachinesEngineView,
    MachinesMacView,
    MachinesVM,
    OverviewNeed,
    PortsPort,
    PortsSharedPort,
} from '@jterrazz/station-api';

import { SEPARATOR } from '../separator';
import type { Tone } from '../tone';
import { wordOf } from './engine-words';
import {
    engineStateOf,
    engineViewStateOf,
    machineKindOf,
    macStateOf,
    shareEndOf,
    shareEndWordOf,
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
    DIFFERING_ACCOUNT_ASKED_WORDS,
    DIFFERING_ACCOUNT_SEEN_WORDS,
    DIFFERING_PORT_WORDS,
    ENGINE_STATE_WORDS,
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
    VM_CHECK_WORDS,
    VM_STATE_WORDS,
} from './words';
import type { OwnState } from './words';

export type Shown = { tone: Tone; word: string };

export function ownState(state: OwnState): Shown {
    return { tone: OWN_STATE_TONES[state], word: OWN_STATE_WORDS[state] };
}

export function vmState(vm: MachinesVM): Shown {
    if (vm.Birth !== null) {
        const { Tone, Word } = wordOf('MachinesBirthStep', vm.Birth.step);
        return { tone: Tone, word: Word };
    }
    const state = vmStateOf(vm);
    if (state === undefined) {
        return { tone: 'plain', word: vm.State === '' ? NO_STATE_WORD : vm.State };
    }
    return { tone: VM_STATE_TONES[state], word: VM_STATE_WORDS[state] };
}

export function birthSentence(birth: MachinesBirth): string {
    const { Word, Detail = '' } = wordOf('MachinesBirthStep', birth.step);
    return `${Word}${SEPARATOR}${Detail} ${birth.position} of ${birth.steps}`;
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

export function vmLifetimeWord(vm: Pick<MachinesVM, 'Lifetime'>): string {
    return wordOf('MachinesLifetime', vm.Lifetime).Word;
}

export function vmKindWord(vm: Pick<MachinesVM, 'Lifetime'>): string {
    return machineKindWord(machineKindOf(vm));
}

export function verdictShown(verdict: CoreVerdictVerdict): Shown {
    return { tone: verdict.Tone, word: verdict.Sentence };
}

export function fileJoinOf(tone: Tone): string {
    return tone === 'ok' ? ' ' : ' from ';
}

export function chapterNoun(chapter: MachinefileChapter, count: number): string {
    const { One, Other } = wordOf('MachinefileChapter', chapter);
    return (count === 1 ? One : Other) ?? chapter;
}

export function engineViewWord(engine: MachinesEngineView): string | undefined {
    const state = engineViewStateOf(engine);
    return state === undefined ? undefined : ENGINE_STATE_WORDS[state];
}

export type SharedChapter = Extract<MachinefileChapter, 'accounts' | 'ports'>;

export type DifferingSide = 'asked' | 'seen';

function differingWordsOf(chapter: SharedChapter, side: DifferingSide) {
    if (chapter === 'ports') {
        return DIFFERING_PORT_WORDS;
    }
    return side === 'asked' ? DIFFERING_ACCOUNT_ASKED_WORDS : DIFFERING_ACCOUNT_SEEN_WORDS;
}

export function differingSide(
    chapter: SharedChapter,
    side: DifferingSide,
    differing: MachinefileDiffering,
): Shown {
    const code = side === 'asked' ? differing.Asked : differing.Seen;
    return { tone: 'plain', word: differingWordsOf(chapter, side)[code] ?? code };
}

export function shareToWord(to: NonNullable<PortsSharedPort['to']>): string {
    return SHARE_TO_WORDS[to];
}
