import { AppIcon, Button, Panel, Row } from '../../design/index';
import { looksSecret } from '../../design/secret';
import { useRead, useWatch } from '../../engine/react';
import { holdsFor } from '../../engine/shown';
import { targetOf } from '../../model/machines';
import type { Found } from '../../model/machines';
import { machineRoute } from '../../shell/route';
import type { Route, Tab } from '../../shell/route';
import { Unanswered } from '../kit/readings';
import { portsOf } from '../ports/ports-of';
import { SharedRow } from '../ports/shared-row';

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
        thisMac: found.mac.ThisMac,
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

export function SharedAccounts({ found, vm, go }: Props) {
    const target = targetOf({ mac: found.mac });
    const accounts = useWatch('accounts.read', { VMs: true }, target);
    const all = <AllOf go={go} label="All sign-ins" tab="Sign-ins" vm={vm} />;
    if (accounts.kind === 'reading') {
        return null;
    }
    if (accounts.kind === 'failed') {
        return (
            <Panel action={all} title="Accounts shared with it">
                <Unanswered error={accounts.error} size="panel" />
            </Panel>
        );
    }
    const report = accounts.out;
    const shares = (report.Shares ?? []).filter((share) => share.vm === vm);
    return (
        <Panel action={all} title="Accounts shared with it">
            {shares.length === 0 ? (
                <p className={styles.none}>No account is shared with it.</p>
            ) : null}
            {shares.map((share) => {
                const withheld = looksSecret(share.account);
                return (
                    <Row
                        key={`${share.vm}/${share.account}`}
                        lead={<AppIcon name={withheld ? '' : share.account} />}
                        state={{ ...holdsFor(share.until, report.Now), asWritten: true }}
                        title={withheld ? 'a share whose name is withheld' : share.account}
                    />
                );
            })}
        </Panel>
    );
}
