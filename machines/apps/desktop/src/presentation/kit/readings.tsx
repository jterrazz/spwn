import type { CoreVerdictVerdict, OverviewSummary } from '@jterrazz/station-api';
import type { ReactNode } from 'react';

import type { Reach } from '../../application/use-cases/reach';
import type { EngineError } from '../../domain/engine-error';
import { clock, sentence } from '../../domain/format';
import { SEPARATOR } from '../../domain/separator';
import type { Tone } from '../../domain/tone';
import { unlessCommand } from '../ui/command';
import { CopyButton } from '../ui/copy-button';
import { asSentence, Button, capitalised, Missing, Panel, StateMark, Verdict } from '../ui/index';
import { shownOrWithheld } from '../ui/secret';
import { failedState } from '../words/failed';
import { ownState, verdictShown } from '../words/shown';

import styles from '../ui/page.module.css';

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
    verdict,
    size,
}: {
    summary: Pick<OverviewSummary, 'Engine' | 'Read'>;
    verdict: CoreVerdictVerdict;
    size: 'page' | 'tab';
}) {
    const { Engine, Read } = summary;
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

type Said = { word: string; tone: Tone; message: string; hint: string };

function saidOf(error: EngineError): Said {
    const { wire, message } = error;
    if (!wire) {
        return {
            ...failedState(error),
            hint: unlessCommand(shownOrWithheld(message)),
            message: 'No answer from the engine',
        };
    }
    return {
        ...failedState(error),
        hint: unlessCommand(shownOrWithheld(wire.hint ?? '')),
        message: shownOrWithheld(wire.message),
    };
}

export function Unanswered({
    error,
    size,
    action,
}: {
    error: EngineError;
    size: 'page' | 'panel' | 'tab';
    action?: ReactNode;
}) {
    const said = saidOf(error);
    const command = error.wire?.fix?.Command ?? '';
    const fix =
        action ??
        (command === '' ? undefined : (
            <CopyButton label="Copy the command" name="Copy the command" text={command} />
        ));
    if (size === 'panel') {
        return (
            <div className={styles.unanswered}>
                <StateMark tone={said.tone} word={said.word} />
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

const UNKNOWN_SAID =
    'No answer came before the connection ended: the engine may have done it. The page below shows what it now says.';

export function Unknown({ error, onDismiss }: { error: EngineError; onDismiss: () => void }) {
    return (
        <span className={styles.outcome}>
            <span className={styles.unanswered}>
                <StateMark {...ownState('unknown')} />
                <span className={styles.said}>
                    {UNKNOWN_SAID}
                    <span className={styles.hint}>{shownOrWithheld(error.message)}</span>
                </span>
            </span>
            <Button onClick={onDismiss} variant="link">
                Dismiss
            </Button>
        </span>
    );
}
