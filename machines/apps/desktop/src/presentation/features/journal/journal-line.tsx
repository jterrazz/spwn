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
import { useGesture } from '../../dependencies/use-gesture';
import { GestureButton, GestureRow } from '../../kit/gestures';
import type { Route } from '../../navigation/route';
import { Row, StatePill } from '../../ui/index';
import type { RowProps } from '../../ui/row';
import { isCalm } from '../../ui/state-pill';
import { updateInputOf } from '../../words/provisional';
import { engineShown } from '../../words/shown';
import { changeOf, endOf, routeOfLine } from './model';

import styles from './changes.module.css';

const RETRY_REREADS: readonly CallName[] = ['overview.read', 'apps.list'];

type Go = (to: Route) => void;

type LineProps = { line: CoreJournalReadLine; macs: readonly MachinesMacView[]; go: Go };

function Tail({ line }: { line: CoreJournalReadLine }) {
    const shown = engineShown('CoreJournalOutcome', line.Outcome);
    const end = endOf(line);
    return (
        <span className={styles.tail}>
            {isCalm(shown.tone) || shown.tone === 'warn' ? null : <StatePill {...shown} />}
            {shown.tone === 'warn' ? <span className={styles.end}>{shown.word}</span> : null}
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
