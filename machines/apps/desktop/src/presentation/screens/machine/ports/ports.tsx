import type { PortsPort, Target } from '@jterrazz/station-api';

import { targetOf } from '../../../../domain/machines/machines';
import type { Found } from '../../../../domain/machines/machines';
import { useRead, useWatch } from '../../../dependencies/dependencies';
import { Needs } from '../../../features/overview/needs-panel';
import { ChapterDiffers } from '../../../kit/chapter-differs';
import { GestureButton, GestureRow, HeldOutcomes } from '../../../kit/gestures';
import { Unanswered, VerdictHead } from '../../../kit/readings';
import { Skeleton, SkeletonPanel } from '../../../kit/skeleton';
import type { Route } from '../../../navigation/route';
import { Panel, Row, StateMark, Value, VerdictSkeleton } from '../../../ui/index';
import { Page } from '../../../ui/page';
import { isDeclarable, isPortShared } from '../../../words/provisional';
import { portReach, shareToWord } from '../../../words/shown';
import { PORT_ANSWER_TONES } from '../../../words/tones';
import { PORT_ANSWER_WORDS } from '../../../words/words';
import { usePortDeclare, usePortShare } from './port-gestures';
import { aboutPort, needsOf, portsOf } from './ports-of';
import type { PortsOf } from './ports-of';
import { SharedRow } from './shared-row';

import styles from './ports.module.css';

const SHARED_TITLE = 'Shared';
const MAC_PORTS_TITLE = 'What listens on this Mac';

function Answer({ port }: { port: PortsPort }) {
    if (port.Answer === '') {
        return null;
    }
    return (
        <StateMark tone={PORT_ANSWER_TONES[port.Answer]} word={PORT_ANSWER_WORDS[port.Answer]} />
    );
}

type ShareableProps = { port: PortsPort; vm: string; target: Target | undefined };

function ShareableRow({ port, vm, target }: ShareableProps) {
    const share = usePortShare({ port: port.Port, target, vm });
    return (
        <GestureRow
            gesture={share}
            lead={<Value look="raised">{`:${port.Port}`}</Value>}
            leadSize="value"
            sentence={aboutPort(port)}
            title={port.Process === '' ? 'process not given yet' : port.Process}
            trail={
                <span className={styles.trail}>
                    <Answer port={port} />
                    <GestureButton
                        gesture={share}
                        label="Share"
                        name={`Share :${port.Port}`}
                        title={`Share ${vm}:${port.Port}?`}
                    />
                </span>
            }
            trailSize="wide"
        />
    );
}

type MacPortProps = { port: PortsPort; mac: string; target: Target | undefined };

function macPortRow(port: PortsPort) {
    return {
        lead: <Value look="raised">{`:${port.Port}`}</Value>,
        leadSize: 'value',
        sentence: aboutPort(port),
        title: port.Process === '' ? 'process not given yet' : port.Process,
        trailSize: 'wide',
    } as const;
}

function DeclarableRow({ port, mac, target }: MacPortProps) {
    const declare = usePortDeclare({ machine: port.Machine.Name, port: port.Port, target });
    const audience = shareToWord('public').toLowerCase();
    return (
        <GestureRow
            {...macPortRow(port)}
            gesture={declare}
            trail={
                <span className={styles.trail}>
                    <Answer port={port} />
                    <StateMark {...portReach(port)} />
                    <GestureButton
                        gesture={declare}
                        label={`Declare it ${audience}`}
                        name={`Declare :${port.Port} ${audience}`}
                        title={`Declare ${port.Machine.Name === '' ? mac : port.Machine.Name}:${port.Port} ${audience}?`}
                    />
                </span>
            }
        />
    );
}

export function MacPortRow({ port, mac, target }: MacPortProps) {
    if (isDeclarable(port)) {
        return <DeclarableRow mac={mac} port={port} target={target} />;
    }
    return (
        <Row
            {...macPortRow(port)}
            trail={
                <span className={styles.trail}>
                    <Answer port={port} />
                    <StateMark {...portReach(port)} />
                </span>
            }
        />
    );
}

type Go = (to: Route) => void;

function Alarm({ found, name, go }: { found: Found; name: string; go: Go }) {
    const target = targetOf({ mac: found.mac });
    const overview = useWatch('overview.read', {}, target);
    if (overview.kind !== 'read') {
        return null;
    }
    return (
        <Needs
            go={go}
            macs={[found.mac]}
            needs={needsOf(overview.out.Summary.Needs ?? [], name)}
            on={name}
            title="Needs you, on its ports"
        />
    );
}

function portName(of: PortsOf, port: number): string {
    const share = of.shared.find((one) => one.guest_port === port);
    if (share !== undefined) {
        return (share.label ?? '') === '' ? share.process : (share.label ?? '');
    }
    return of.ports.find((one) => one.Port === port)?.Process ?? '';
}

type VmPortsProps = { of: PortsOf; vm: string; target: Target | undefined };

function VmPorts({ of, vm, target }: VmPortsProps) {
    const listening = of.ports.filter((port) => !isPortShared(port, of.shared));
    const rest = listening.length + of.notShared.length;
    return (
        <>
            {of.shared.length === 0 && rest > 0 ? null : (
                <Panel title={SHARED_TITLE}>
                    {of.shared.length === 0 ? (
                        <p className={styles.empty}>Nothing listed on {vm}.</p>
                    ) : (
                        of.shared.map((share) => (
                            <SharedRow
                                key={`${share.vm}:${share.guest_port}`}
                                onThisMac={of.mac?.this_mac === true}
                                share={share}
                                target={target}
                            />
                        ))
                    )}
                </Panel>
            )}
            {rest === 0 ? null : (
                <Panel title="Listening, not shared">
                    {listening.map((port) => (
                        <ShareableRow key={port.Port} port={port} target={target} vm={vm} />
                    ))}
                    {of.notShared.map((row) => (
                        <Row
                            key={`not-shared/${row.vm}`}
                            size="slim"
                            title={row.detail}
                            value={String(row.count)}
                        />
                    ))}
                </Panel>
            )}
        </>
    );
}

type MacPortsProps = { of: PortsOf; name: string; target: Target | undefined };

function MacPorts({ of, name, target }: MacPortsProps) {
    return (
        <Panel title={MAC_PORTS_TITLE}>
            {of.ports.length === 0 ? (
                <p className={styles.empty}>
                    {of.mac ? `Nothing listed on ${name}.` : `The answer names no Mac ${name}.`}
                </p>
            ) : (
                of.ports.map((port) => (
                    <MacPortRow key={port.Port} mac={name} port={port} target={target} />
                ))
            )}
        </Panel>
    );
}

export function PortsTab({ found, go }: { found: Found; go: Go }) {
    const name = found.vm?.Name ?? found.mac.Name;
    const listing = useRead('ports.list', { Of: name });
    if (listing.kind === 'failed') {
        return (
            <Page>
                <Unanswered error={listing.error} size="tab" />
            </Page>
        );
    }
    const vm = found.vm?.Name;
    if (listing.kind === 'reading') {
        return (
            <Page>
                <Skeleton what={`the ports of ${name}`}>
                    <VerdictSkeleton size="tab" />
                    <Alarm found={found} go={go} name={name} />
                    <SkeletonPanel
                        lead="value"
                        title={vm === undefined ? MAC_PORTS_TITLE : SHARED_TITLE}
                        trail="wide"
                    />
                </Skeleton>
            </Page>
        );
    }
    const of = portsOf(listing.out, { mac: found.mac.Name, vm });
    return (
        <Page>
            <VerdictHead size="tab" verdict={listing.out.verdict} />
            <Alarm found={found} go={go} name={name} />
            <HeldOutcomes>
                {vm === undefined ? (
                    <MacPorts name={name} of={of} target={targetOf({ mac: found.mac })} />
                ) : (
                    <VmPorts of={of} target={targetOf({ mac: found.mac })} vm={vm} />
                )}
            </HeldOutcomes>
            {vm === undefined ? null : (
                <ChapterDiffers
                    chapter="ports"
                    found={found}
                    nameOf={(differing) => portName(of, differing.Port)}
                />
            )}
        </Page>
    );
}
