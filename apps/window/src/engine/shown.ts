import type {
    AppsReport,
    CatalogAudience,
    CatalogKind,
    CoreVerdictVerdict,
    EngineReading,
    MachinefileChapter,
    MachinefileDiffering,
    MachinesEngineView,
    MachinesMacView,
    MachinesVM,
    OverviewNeed,
    PortsPort,
    StationWord,
} from '@jterrazz/station-api';

import type { Tone } from '../tone';
import { wordOf } from './engine-words';
import type { CodeOf, Worded } from './engine-words';
import {
    accountSideOf,
    chapterOfKind,
    engineViewStateOf,
    machineKindOf,
    macStateOf,
    shareEndOf,
    shareEndWordOf,
} from './provisional';
import {
    APP_WHY_TONES,
    MAC_STATE_TONES,
    NEWEST_TONE,
    OWN_STATE_TONES,
    PORT_REACH_TONES,
    REVERSIBLE_TONE,
    SHARE_END_TONES,
} from './tones';
import {
    APP_WHY_WORDS,
    DIFFERING_WORDS,
    MAC_STATE_WORDS,
    MACHINE_KIND_WORDS,
    NEWEST_WORD,
    newestOutWord,
    NO_STATE_WORD,
    OWN_STATE_WORDS,
    PORT_REACH_WORDS,
    REVERSIBLE_WORD,
    SSH_WORDS,
} from './words';
import type { OwnState } from './words';

export type Shown = { tone: Tone; word: string };

export function ownState(state: OwnState): Shown {
    return { tone: OWN_STATE_TONES[state], word: OWN_STATE_WORDS[state] };
}

// invariant: the one place an engine's tone becomes a tone the design draws: the five are the design's own five.
function toneOf(said: StationWord): Tone {
    return said.Tone;
}

export function engineShown<E extends Worded>(of: E, code: CodeOf<E>): Shown {
    const said = wordOf(of, code);
    return { tone: toneOf(said), word: said.Word };
}

export function vmState(vm: MachinesVM): Shown {
    return vm.Birth === null
        ? engineShown('MachinesVMState', vm.State)
        : engineShown('MachinesBirthStep', vm.Birth.step);
}

export function vmCheck(vm: MachinesVM): Shown {
    const { state } = vm.CheckStatus;
    return state === undefined
        ? { tone: 'plain', word: NO_STATE_WORD }
        : engineShown('MachinesCheckState', state);
}

export function vmSsh(vm: MachinesVM): Shown {
    return { tone: 'plain', word: vm.SSH ? SSH_WORDS.answers : SSH_WORDS.silent };
}

export function macState(mac: MachinesMacView): Shown {
    const state = macStateOf(mac);
    return { tone: MAC_STATE_TONES[state], word: MAC_STATE_WORDS[state] };
}

export function needState(need: OverviewNeed): Shown {
    return engineShown('CoreFindingSeverity', need.Severity);
}

export function engineState(reading: EngineReading): Shown {
    return engineShown('EngineReadState', reading.State);
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
    return state === undefined ? undefined : wordOf('EngineReadState', state).Word;
}

export type SharedChapter = Extract<MachinefileChapter, 'accounts' | 'ports'>;

export type DifferingSide = 'asked' | 'seen';

function portSide(to: CatalogAudience | undefined): Shown {
    return to === undefined
        ? { tone: 'plain', word: DIFFERING_WORDS['not-shared'] }
        : engineShown('CatalogAudience', to);
}

export function differingSide(
    chapter: SharedChapter,
    side: DifferingSide,
    differing: MachinefileDiffering,
): Shown {
    if (chapter === 'ports') {
        return portSide(side === 'asked' ? differing.AskedTo : differing.SeenTo);
    }
    return { tone: 'plain', word: DIFFERING_WORDS[accountSideOf(side, differing)] };
}

export function shareToWord(to: CatalogAudience): string {
    return wordOf('CatalogAudience', to).Word;
}

// value: an app is the plain kind, drawn by no chip: a row draws the kind of a manager or a setting alone.
export function kindChipWord(kind: CatalogKind): string {
    return kind === 'app' ? '' : wordOf('CatalogKind', kind).Word;
}

export function shelfWord(kind: CatalogKind): string {
    return wordOf('MachinefileChapter', chapterOfKind(kind)).Word;
}
