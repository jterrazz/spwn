import type { CoreVerdictVerdict, OverviewSummary } from '@jterrazz/station-api';
import type { ReactNode } from 'react';

import type { Reach } from '../../application/use-cases/reach';
import { CopyButton } from '../../design/copy-button';
import {
    asSentence,
    capitalised,
    Missing,
    Panel,
    StatePill,
    Value,
    Verdict,
} from '../../design/index';
import { shownOrWithheld } from '../../design/secret';
import type { EngineError } from '../../domain/engine-error';
import { clock, fixLine, sentence } from '../../domain/format';
import { SEPARATOR } from '../../domain/separator';
import type { Tone } from '../../domain/tone';
import { failedState } from '../../engine/failed';
import { ownState, verdictShown } from '../../engine/shown';

import styles from '../page.module.css';

export function NoVerdict({ size, line }: { size: 'page' | 'tab'; line?: ReactNode }) {
    return (
        <Verdict
            line={
                line ?? 'What follows is what the engine answers, with no sentence that sums it up'
            }
            size={size}
            title="No verdict from the engine."
            {...ownState('not-given')}
        />
    );
}

export function VerdictHead({
    verdict,
    size,
    action,
}: {
    verdict: CoreVerdictVerdict;
    size: 'page' | 'tab';
    action?: ReactNode;
}) {
    return (
        <Verdict
            action={action}
            line={verdict.Line}
            size={size}
            title={asSentence(verdict.Sentence)}
            {...verdictShown(verdict)}
        />
    );
}

export function SummaryHead({
    summary,
    size,
}: {
    summary: Pick<OverviewSummary, 'Engine' | 'Read' | 'Verdict'>;
    size: 'page' | 'tab';
}) {
    const { Engine, Read, Verdict: verdict } = summary;
    const read = Engine === 'stopped' ? clock(Read) : undefined;
    return (
        <Verdict
            line={sentence([verdict.Line, read === undefined ? undefined : `Last read at ${read}`])}
            size={size}
            title={asSentence(verdict.Sentence)}
            {...verdictShown(verdict)}
        />
    );
}

export function Reading({ size, what }: { size: 'page' | 'tab'; what: string }) {
    return (
        <Verdict
            line="The engine has not answered yet"
            size={size}
            title={`Reading ${what}…`}
            {...ownState('reading')}
        />
    );
}

type Said = { word: string; tone: Tone; message: string; hint: string; fix: string };

function saidOf(error: EngineError): Said {
    const { wire, message } = error;
    if (!wire) {
        return {
            ...failedState(error),
            fix: '',
            hint: shownOrWithheld(message),
            message: 'No answer from the engine',
        };
    }
    return {
        ...failedState(error),
        fix: fixLine(wire.fix),
        hint: shownOrWithheld(wire.hint ?? ''),
        message: shownOrWithheld(wire.message),
    };
}

export function Unanswered({
    error,
    size,
}: {
    error: EngineError;
    size: 'page' | 'panel' | 'tab';
}) {
    const said = saidOf(error);
    const command = error.wire?.fix?.Command ?? '';
    const copied = command === '' ? said.fix : command;
    const fix =
        said.fix === '' ? undefined : (
            <span className={styles.fix}>
                <Value>{said.fix}</Value>
                {shownOrWithheld(said.fix) === said.fix ? (
                    <CopyButton name={`Copy ${copied}`} text={copied} />
                ) : null}
            </span>
        );
    if (size === 'panel') {
        return (
            <div className={styles.unanswered}>
                <StatePill tone={said.tone} word={said.word} />
                <span className={styles.said}>
                    {asSentence(said.message)}
                    {said.hint === '' ? null : <span className={styles.hint}>{said.hint}</span>}
                </span>
                {fix}
            </div>
        );
    }
    return (
        <Verdict
            action={fix}
            line={sentence([capitalised(said.word), said.hint])}
            size={size}
            title={asSentence(said.message)}
            tone={said.tone}
            word={said.word}
        />
    );
}

export function Absent({ reach }: { reach: Extract<Reach, { kind: 'unanswered' }> }) {
    return (
        <Verdict
            line={`${reach.reason}${SEPARATOR}trying again in ${Math.round(reach.retryIn / 1000)} s`}
            size="page"
            title={asSentence(reach.said)}
            {...ownState('engine-absent')}
        />
    );
}

export function NotGiven({
    title,
    what,
    action,
}: {
    title: string;
    what: string;
    action?: ReactNode;
}) {
    return (
        <Panel action={action} title={title}>
            <Missing>{what}</Missing>
        </Panel>
    );
}
