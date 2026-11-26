import type {
    AppsUpdateInput,
    CallName,
    CoreJournalReadLine,
    MachinesMacView,
    Target,
} from '@jterrazz/station-api';
import type { ReactNode } from 'react';

import { clock } from '../../../domain/format';
import { findMachine, targetOf } from '../../../domain/machines/machines';
import type { Tone } from '../../../domain/tone';
import { useGesture } from '../../dependencies/use-gesture';
import { GestureButton, GestureRow } from '../../kit/gestures';
import type { Route } from '../../navigation/route';
import { Row, StateMark } from '../../ui/index';
import type { RowProps } from '../../ui/row';
import { updateInputOf } from '../../words/provisional';
import { engineShown } from '../../words/shown';
import { changeOf, endOf, routeOfLine } from './model';

import styles from './changes.module.css';

const RETRY_REREADS: readonly CallName[] = ['overview.read', 'apps.list'];

type Go = (to: Route) => void;

type LineProps = { line: CoreJournalReadLine; macs: readonly MachinesMacView[]; go: Go };

// invariant: a past event that went well draws nothing: a held or failed line draws its mark and its word.
function wentWrong(tone: Tone): boolean {
    return tone === 'warn' || tone === 'bad';
}

function Tail({ line }: { line: CoreJournalReadLine }) {
    const shown = engineShown('CoreJournalOutcome', line.Outcome);
    const end = endOf(line);
    return (
        <span className={styles.tail}>
            {wentWrong(shown.tone) ? <StateMark said="after" {...shown} /> : null}
            {end === '' ? null : <span className={styles.end}>{end}</span>}
        </span>
    );
}

function rowOf({ line, go }: LineProps, link?: ReactNode): RowProps {
    const route = routeOfLine(line);
    const change = changeOf(line);
    const row: RowProps = {
        lead: <span className={styles.at}>{clock(line.At) ?? ''}</span>,
        leadSize: 'clock',
        size: 'slim',
        title: line.Text,
        titleLook: 'strong',
        value: <Tail line={line} />,
        valueSize: 'long',
        ...(change === '' ? {} : { tag: change }),
        ...(link === undefined ? {} : { link }),
    };
    if (route === undefined) {
        return row;
    }
    return {
        ...row,
        onOpen: () => {
            go(route);
        },
    };
}

type Retrying = { input: AppsUpdateInput; target: Target | undefined; verb: string };

function retryingOf({ line, macs }: LineProps): Retrying | undefined {
    const { Retry } = line;
    if (Retry === null) {
        return undefined;
    }
    const input = updateInputOf(Retry);
    const found = findMachine(macs, Retry.On);
    if (input === undefined || found === undefined) {
        return undefined;
    }
    return { input, target: targetOf(found), verb: Retry.Verb };
}

function Retried({ retrying, ...props }: LineProps & { retrying: Retrying }) {
    const { input, target, verb } = retrying;
    const gesture = useGesture('apps.update', {
        input,
        rereads: RETRY_REREADS,
        subject: props.line.Text,
        target,
    });
    return (
        <GestureRow
            gesture={gesture}
            {...rowOf(
                props,
                <GestureButton
                    gesture={gesture}
                    label={verb}
                    name={`${verb} ${props.line.Text}`}
                    title={`${verb} ${props.line.Text}?`}
                />,
            )}
        />
    );
}

export function JournalLineRow(props: LineProps) {
    const retrying = retryingOf(props);
    if (retrying === undefined) {
        return <Row {...rowOf(props)} />;
    }
    return <Retried {...props} retrying={retrying} />;
}
