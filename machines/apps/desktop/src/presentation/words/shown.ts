import type {
    AccountsProgramRow,
    AppsReport,
    CatalogAudience,
    CatalogKind,
    CoreVerdictVerdict,
    EngineReading,
    MachinefileChapter,
    MachinefileDiffering,
    MachinefileFollower,
    MachinefileStep,
    MachinesEngineView,
    MachinesHold,
    MachinesMacView,
    MachinesRollUp,
    MachinesUptimeCell,
    MachinesVM,
    OverviewNeed,
    PortsPort,
    SettingsRow,
    StationWord,
} from '@jterrazz/station-api';

import type { CoreKind } from '../../application/use-cases/core-flow';
import type { UpdateKind } from '../../application/use-cases/update-flow';
import type { Tone } from '../../domain/tone';
import { wordOf } from './engine-words';
import type { CodeOf, Worded } from './engine-words';
import { machineKindOf } from './provisional';
import {
    APP_WHY_TONES,
    CORE_TONES,
    NEWEST_TONE,
    OWN_STATE_TONES,
    PORT_REACH_TONES,
    REVERSIBLE_TONE,
    UPDATE_TONES,
    UPTIME_MARK_TONES,
} from './tones';
import {
    APP_WHY_WORDS,
    CORE_WORDS,
    MACHINE_KIND_WORDS,
    NEWEST_WORD,
    newestOutWord,
    NO_STATE_WORD,
    NOT_SHARED_WORD,
    OWN_STATE_WORDS,
    PORT_REACH_WORDS,
    REVERSIBLE_WORD,
    SSH_WORDS,
    UPDATE_WORDS,
} from './words';
import type { OwnState } from './words';

export type Shown = { tone: Tone; word: string };

export function ownState(state: OwnState): Shown {
    return { tone: OWN_STATE_TONES[state], word: OWN_STATE_WORDS[state] };
}

export function coreState(kind: CoreKind): Shown {
    return { tone: CORE_TONES[kind], word: CORE_WORDS[kind] };
}

export function updateState(kind: UpdateKind): Shown {
    return { tone: UPDATE_TONES[kind], word: UPDATE_WORDS[kind] };
}

// invariant: the one place an engine's tone becomes a tone the design draws: the five are the design's own five.
function toneOf(said: StationWord): Tone {
    return said.Tone;
}

export function engineShown<E extends Worded>(of: E, code: CodeOf<E>): Shown {
    const said = wordOf(of, code);
    return { tone: toneOf(said), word: said.Word };
}

const NOT_GIVEN: Shown = { tone: 'plain', word: NO_STATE_WORD };

export function vmState(vm: Pick<MachinesVM, 'Standing'>): Shown {
    return vm.Standing === undefined ? NOT_GIVEN : engineShown('MachinesVMStanding', vm.Standing);
}

export function rollUpState({ N, State }: MachinesRollUp): Shown {
    const said = wordOf('MachinesRollUpState', State);
    const noun = N === 1 ? said.One : said.Other;
    return {
        tone: toneOf(said),
        word: N === undefined || noun === undefined ? said.Word : `${String(N)} ${noun}`,
    };
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

export function macState(mac: Pick<MachinesMacView, 'Standing'>): Shown {
    return mac.Standing === undefined
        ? NOT_GIVEN
        : engineShown('MachinesMacStanding', mac.Standing);
}

export function macCardState(mac: MachinesMacView): Shown {
    return mac.Standing === 'online' && mac.Machines !== undefined
        ? rollUpState(mac.Machines)
        : macState(mac);
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

export function settingState(state: SettingsRow['State']): Shown {
    return engineShown('SettingsState', state);
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

// value: a count of the wire's own rows, the ones whose How the engine says is granted; no judgement is made here.
export function grantedCount(programs: readonly Pick<AccountsProgramRow, 'How'>[]): number {
    return programs.filter((row) => row.How === 'granted').length;
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

export function followerState({
    Differences,
    State,
}: Pick<MachinefileFollower, 'Differences' | 'State'>): Shown {
    const said = wordOf('MachinefileFollowerState', State);
    if (State !== 'differs') {
        return { tone: toneOf(said), word: said.Word };
    }
    const { One, Other } = said;
    return {
        tone: toneOf(said),
        word: `${String(Differences)} ${(Differences === 1 ? One : Other) ?? said.Word}`,
    };
}

export function fileJoinOf(tone: Tone): string {
    return tone === 'ok' ? ' ' : ' from ';
}

export function chapterNoun(chapter: MachinefileChapter, count: number): string {
    const { One, Other } = wordOf('MachinefileChapter', chapter);
    return (count === 1 ? One : Other) ?? chapter;
}

export function engineViewStopped({ State }: MachinesEngineView): boolean {
    return State === 'stopped';
}

// value: the engine's sentence of a hold whose state words.json tones warn or bad, as sent; a calm hold says nothing.
export function holdAgainst(hold: MachinesHold | undefined): string | undefined {
    if (hold === undefined) {
        return undefined;
    }
    const { tone } = engineShown('MachinesHoldState', hold.State);
    return tone === 'warn' || tone === 'bad' ? hold.Sentence : undefined;
}

export function dayOpensHistory({ Mark }: MachinesUptimeCell): boolean {
    const tone = UPTIME_MARK_TONES[Mark];
    return tone === 'warn' || tone === 'bad';
}
export type SharedChapter = Extract<MachinefileChapter, 'accounts' | 'ports'>;

export type DifferingSide = 'asked' | 'seen';

function portSide(to: CatalogAudience | undefined): Shown {
    return to === undefined
        ? { tone: 'plain', word: NOT_SHARED_WORD }
        : engineShown('CatalogAudience', to);
}

export function differingLeftOut(
    chapter: SharedChapter,
    side: DifferingSide,
    differing: MachinefileDiffering,
): boolean {
    return chapter === 'accounts' && side === 'asked' && differing.AskedAs === 'not-in-the-file';
}

export function differingSide(
    chapter: SharedChapter,
    side: DifferingSide,
    differing: MachinefileDiffering,
): Shown {
    if (chapter === 'ports') {
        return portSide(side === 'asked' ? differing.AskedTo : differing.SeenTo);
    }
    if (side === 'asked') {
        return differing.AskedAs === undefined
            ? { tone: 'plain', word: NOT_SHARED_WORD }
            : engineShown('MachinefileFileSide', differing.AskedAs);
    }
    return differing.SeenAs === undefined
        ? { tone: 'plain', word: NOT_SHARED_WORD }
        : engineShown('MachinefileMacSide', differing.SeenAs);
}

export function shareToWord(to: CatalogAudience): string {
    return wordOf('CatalogAudience', to).Word;
}

// value: an app is the plain kind, drawn by no chip: a row draws the kind of a manager or a setting alone.
export function kindChipWord(kind: CatalogKind): string {
    return kind === 'app' ? '' : wordOf('CatalogKind', kind).Word;
}

export function shelfWord(chapter: MachinefileChapter): string {
    return wordOf('MachinefileChapter', chapter).Word;
}

export function stepBoxed(step: MachinefileStep): string {
    return step.Version === '' ? step.Rule : step.Version;
}
