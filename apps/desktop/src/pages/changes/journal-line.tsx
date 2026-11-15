import type { CallName, MachinesMacView } from '@jterrazz/station-api';
import type { ReactNode } from 'react';

import { clock } from '../../domain/format';
import { findMachine, targetOf } from '../../domain/machines/machines';
import { useGesture } from '../../presentation/dependencies/use-gesture';
import { outcomeShown } from '../../presentation/features/journal/kept-journal/journal';
import type {
    JournalLine,
    JournalRetry,
} from '../../presentation/features/journal/kept-journal/journal';
import type { Route } from '../../presentation/navigation/route';
import { Row, StatePill } from '../../presentation/ui/index';
import type { RowProps } from '../../presentation/ui/row';
import { isCalm } from '../../presentation/ui/state-pill';
import { GestureButton, GestureRow } from '../kit/gestures';
import { changeOf, endOf, routeOfLine } from './model';

import styles from './changes.module.css';

const RETRY_REREADS: readonly CallName[] = ['overview.read', 'apps.list'];

type Go = (to: Route) => void;

type LineProps = { line: JournalLine; macs: readonly MachinesMacView[]; go: Go };

function Tail({ line }: { line: JournalLine }) {
    const shown = outcomeShown(line.Outcome);
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

function Retried({ retry, ...props }: LineProps & { retry: JournalRetry }) {
    const found = findMachine(props.macs, retry.Machine);
    const gesture = useGesture(retry.Call, {
        input: retry.Input,
        rereads: RETRY_REREADS,
        subject: props.line.Text,
        target: found ? targetOf(found) : { name: retry.Machine },
    });
    return (
        <GestureRow
            gesture={gesture}
            {...rowOf(
                props,
                <GestureButton
                    gesture={gesture}
                    label={retry.Verb}
                    name={`${retry.Verb} ${props.line.Text}`}
                    title={`${retry.Verb} ${props.line.Text}?`}
                />,
            )}
        />
    );
}

export function JournalLineRow(props: LineProps) {
    const { Retry } = props.line;
    if (Retry === null) {
        return <Row {...rowOf(props)} />;
    }
    return <Retried {...props} retry={Retry} />;
}
