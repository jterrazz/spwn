import { Button, Panel } from '../../design/index';
import { looksSecret } from '../../design/secret';
import { useRead, useWatch } from '../../engine/react';
import { holdsFor } from '../../engine/shown';
import { targetOf } from '../../model/machines';
import type { Found } from '../../model/machines';
import { machineRoute } from '../../shell/route';
import type { Route, Tab } from '../../shell/route';
import { accountsOf } from '../accounts/accounts-of';
import { ShareRow } from '../accounts/share-gestures';
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
    const accounts = useWatch('accounts.list', { VMs: true }, target);
    const all = <AllOf go={go} label="All accounts" tab="Accounts" vm={vm} />;
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
    const { shares } = accountsOf(report, vm, true);
    return (
        <Panel action={all} title="Accounts shared with it">
            {shares.length === 0 ? (
                <p className={styles.none}>No account is shared with it.</p>
            ) : null}
            {shares.map((share) => (
                <ShareRow
                    account={share.account}
                    holds={holdsFor(share.until, report.Now)}
                    key={`${share.vm}/${share.account}`}
                    lead={undefined}
                    shown={
                        looksSecret(share.account)
                            ? 'a share whose name is withheld'
                            : share.account
                    }
                    target={target}
                    vm={share.vm}
                />
            ))}
        </Panel>
    );
}
