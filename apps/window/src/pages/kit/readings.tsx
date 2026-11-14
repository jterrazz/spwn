import type { ReactNode } from 'react';

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
import { SEPARATOR } from '../../design/separator';
import type { EngineError } from '../../engine/engine-error';
import { failedState } from '../../engine/failed';
import type { Reach } from '../../engine/reach';
import { ownState } from '../../engine/shown';
import { fixLine, sentence } from '../../model/format';
import type { Tone } from '../../tone';

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
