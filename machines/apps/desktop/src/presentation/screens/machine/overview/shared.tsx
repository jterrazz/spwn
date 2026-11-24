import { aimed } from '../../../../domain/machines/aim';
import { targetOf } from '../../../../domain/machines/machines';
import type { Found } from '../../../../domain/machines/machines';
import { useRead, useWatch } from '../../../dependencies/dependencies';
import { Unanswered } from '../../../kit/readings';
import { machineRoute, tabsOf } from '../../../navigation/route';
import type { Route, Tab } from '../../../navigation/route';
import { asSentence, Button, Panel, Row } from '../../../ui/index';
import { portsOf } from '../ports/ports-of';
import { SharedRow } from '../ports/shared-row';
import { rowShown } from '../sign-ins/sign-ins';

import styles from './overview.module.css';

type Go = (to: Route) => void;

type Props = { found: Found; vm: string; go: Go };

function AllOf({ vm, tab, label, go }: { vm: string; tab: Tab; label: string; go: Go }) {
    return (
        <Button
            onClick={() => {
                go(machineRoute(vm, tab));
            }}
            variant="link"
        >
            {label}
        </Button>
    );
}

export function SharedPorts({ found, vm, go }: Props) {
    const listing = useRead('ports.list', { AllMachines: true });
    const all = <AllOf go={go} label="All ports" tab="Ports" vm={vm} />;
    if (listing.kind === 'reading') {
        return null;
    }
    if (listing.kind === 'failed') {
        return (
            <Panel action={all} title="Ports">
                <Unanswered error={listing.error} size="panel" />
            </Panel>
        );
    }
    const { mac, shared } = portsOf(listing.out, {
        mac: found.mac.Name,
        vm,
    });
    return (
        <Panel action={all} title="Ports">
            {shared.length === 0 ? <p className={styles.none}>No port is shared.</p> : null}
            {shared.map((share) => (
                <SharedRow
                    compact
                    key={`${share.vm}:${share.guest_port}`}
                    onThisMac={mac?.this_mac === true}
                    share={share}
                    target={targetOf({ mac: found.mac })}
                />
            ))}
        </Panel>
    );
}

export function SignInsPanel({ found, vm, go }: Props) {
    const { input, target } = aimed(found, {});
    const listed = useWatch('accounts.list', input, target);
    const all = <AllOf go={go} label="All sign-ins" tab="Sign-ins" vm={vm} />;
    if (!tabsOf(found.vm).includes('Sign-ins') || listed.kind === 'reading') {
        return null;
    }
    if (listed.kind === 'failed') {
        return (
            <Panel action={all} title="Sign-ins">
                <Unanswered error={listed.error} size="panel" />
            </Panel>
        );
    }
    const programs = listed.out.Programs ?? [];
    return (
        <Panel action={all} note={String(programs.length)} title="Sign-ins">
            {programs.length === 0 ? (
                <p className={styles.none}>{asSentence(listed.out.Verdict.Sentence)}</p>
            ) : null}
            {programs.map((row) => (
                <Row {...rowShown(row, true)} key={row.Program} />
            ))}
        </Panel>
    );
}
