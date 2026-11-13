import type { PortsPort, Target } from '@jterrazz/station-api';

import { Panel, Row, StatePill, Value } from '../../design/index';
import { isDeclarable, isPortShared } from '../../engine/provisional';
import { useRead, useWatch } from '../../engine/react';
import { portReach } from '../../engine/shown';
import { PORT_ANSWER_TONES } from '../../engine/tones';
import { PORT_ANSWER_WORDS } from '../../engine/words';
import { targetOf } from '../../model/machines';
import type { Found } from '../../model/machines';
import type { Route } from '../../shell/route';
import { GestureButton, GestureRow, HeldOutcomes } from '../kit/gestures';
import { NoVerdict, Reading, Unanswered } from '../kit/readings';
import { Needs } from '../needs/needs-panel';
import { Page } from '../page';
import { usePortDeclare, usePortShare } from './port-gestures';
import { aboutPort, needsOf, portsOf } from './ports-of';
import type { PortsOf } from './ports-of';
import { SharedRow } from './shared-row';

import styles from './ports.module.css';

const MAC_LINE = 'The ports of its machines are on their own Ports tab';

function Answer({ port }: { port: PortsPort }) {
    if (port.Answer === '') {
        return null;
    }
    return (
        <StatePill tone={PORT_ANSWER_TONES[port.Answer]} word={PORT_ANSWER_WORDS[port.Answer]} />
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
    return (
        <GestureRow
            {...macPortRow(port)}
            gesture={declare}
            trail={
                <span className={styles.trail}>
                    <Answer port={port} />
                    <StatePill {...portReach(port)} />
                    <GestureButton
                        gesture={declare}
                        label="Declare it public"
                        name={`Declare :${port.Port} public`}
                        title={`Declare ${port.Machine.Name === '' ? mac : port.Machine.Name}:${port.Port} public?`}
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
                    <StatePill {...portReach(port)} />
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

type VmPortsProps = { of: PortsOf; vm: string; target: Target | undefined };

function VmPorts({ of, vm, target }: VmPortsProps) {
    const listening = of.ports.filter((port) => !isPortShared(port, of.shared));
    const rest = listening.length + of.notShared.length + of.quiet.length;
    return (
        <>
            {of.shared.length === 0 && rest > 0 ? null : (
                <Panel title="Shared">
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
                    {of.quiet.map((row) => (
                        <Row
                            key={`quiet/${row.vm}`}
                            size="slim"
                            title="On ports the system keeps"
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
        <Panel title="What listens on this Mac">
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

function lineOf(of: PortsOf, vm: string | undefined): string {
    const range = of.mac?.range;
    if (vm === undefined) {
        return MAC_LINE;
    }
    return range
        ? `It may share ports ${range.low} to ${range.high} with your devices`
        : 'Its ports are shared from its Mac';
}

export function PortsTab({ found, go }: { found: Found; go: Go }) {
    const name = found.vm?.Name ?? found.mac.Name;
    const listing = useRead('ports.list', { AllMachines: true });
    if (listing.kind === 'failed') {
        return (
            <Page>
                <Unanswered error={listing.error} size="tab" />
            </Page>
        );
    }
    if (listing.kind === 'reading') {
        return (
            <Page>
                <Reading size="tab" what={`the ports of ${name}`} />
            </Page>
        );
    }
    const vm = found.vm?.Name;
    const of = portsOf(listing.out, { mac: found.mac.Name, thisMac: found.mac.ThisMac, vm });
    return (
        <Page>
            <NoVerdict line={lineOf(of, vm)} size="tab" />
            <Alarm found={found} go={go} name={name} />
            <HeldOutcomes>
                {vm === undefined ? (
                    <MacPorts name={name} of={of} target={targetOf({ mac: found.mac })} />
                ) : (
                    <VmPorts of={of} target={targetOf({ mac: found.mac })} vm={vm} />
                )}
            </HeldOutcomes>
        </Page>
    );
}
