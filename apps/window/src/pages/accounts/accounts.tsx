import type {
    AccountsAccount,
    AccountsReport,
    AccountsShared,
    Target,
} from '@jterrazz/station-api';

import { CopyButton } from '../../design/copy-button';
import { AppIcon, Button, MachineChip, Panel, Row, StatePill, Value } from '../../design/index';
import { looksSecret } from '../../design/secret';
import { SEPARATOR } from '../../design/separator';
import { Openable } from '../../engine/opener';
import { machineKindOf } from '../../engine/provisional';
import { useWatch } from '../../engine/react';
import { holdsFor } from '../../engine/shown';
import { ACCOUNT_STATE_TONES } from '../../engine/tones';
import { sentence } from '../../model/format';
import { targetOf } from '../../model/machines';
import type { Found } from '../../model/machines';
import { machineRoute } from '../../shell/route';
import type { Route } from '../../shell/route';
import { ChapterDiffers } from '../kit/chapter-differs';
import { GestureRow, HeldOutcomes } from '../kit/gestures';
import { useIdentityOf } from '../kit/identity';
import { OpenButton } from '../kit/opens';
import { NoVerdict, Reading, Unanswered } from '../kit/readings';
import { Page } from '../page';
import { aboutOf, accountsOf, sharesOf, whoOf, wordOf } from './accounts-of';
import { ScreenedOutcome, ShareRow, TakeBackButton, useTakeBack } from './share-gestures';

import styles from './accounts.module.css';

const DURATIONS = ['4 h', '12 h', '7 days'] as const;

const NO_FIX = 'The engine gives no command that signs in';

function elsewhere(mac: string): string {
    return `Sign in on ${mac} itself: the terminal the window opens is this Mac's`;
}

const MAC_LINE = `An account lives on this Mac; a machine gets a copy for a time, never the account itself${SEPARATOR}no secret is ever shown here`;

type Go = (to: Route) => void;

function Withheld({ fields }: { fields: readonly string[] }) {
    if (fields.length === 0) {
        return null;
    }
    return (
        <p className={styles.withheld}>
            Not shown: {fields.join(', ')} held what looks like a secret, and a secret is never
            drawn here.
        </p>
    );
}

function Holder({ found, name, go }: { found: Found; name: string; go: Go }) {
    const known = (found.mac.VMs ?? []).find((candidate) => candidate.Name === name);
    const identityOf = useIdentityOf();
    if (!known) {
        return <Value>{name}</Value>;
    }
    return (
        <MachineChip
            identity={identityOf(name)}
            kind={machineKindOf(known)}
            name={name}
            onOpen={() => {
                go(machineRoute(name, 'Accounts'));
            }}
        />
    );
}

function SharedWithChips({
    shares,
    found,
    go,
}: {
    shares: readonly AccountsShared[];
    found: Found;
    go: Go;
}) {
    if (shares.length === 0) {
        return null;
    }
    return (
        <p className={styles.sharedWith}>
            <span className={styles.hint}>shared with</span>
            {shares.map((share) => (
                <Holder found={found} go={go} key={share.vm} name={share.vm} />
            ))}
        </p>
    );
}

function SignIn({ hint }: { hint: string }) {
    if (hint === '' || looksSecret(hint)) {
        return null;
    }
    return (
        <>
            <Value>{hint}</Value>
            <CopyButton name={`Copy ${hint}`} text={hint} />
        </>
    );
}

type AccountRowProps = {
    account: AccountsAccount;
    report: AccountsReport;
    found: Found;
    go: Go;
    target: Target | undefined;
};

function AccountRow({ account, report, found, go, target }: AccountRowProps) {
    const about = aboutOf(account);
    const loan = account.shared;
    const hint = account.hint ?? '';
    const on = found.mac.Name;
    const shown = account.provider;
    const back = useTakeBack({ account: account.provider, shown, target, vm: on });
    const shares = sharesOf(report, account.provider);
    return (
        <>
            <GestureRow
                below={<SharedWithChips found={found} go={go} shares={shares} />}
                gesture={back}
                lead={<AppIcon name={account.provider} />}
                outcome={<ScreenedOutcome gesture={back} />}
                sentence={sentence(about.shown)}
                title={account.provider}
                trail={
                    <span className={styles.trail}>
                        <SignIn hint={hint} />
                        {loan ? <StatePill {...holdsFor(loan.until, report.Now)} /> : null}
                        <StatePill
                            tone={ACCOUNT_STATE_TONES[account.state]}
                            word={wordOf(account)}
                        />
                        {hint === '' || looksSecret(hint) ? null : (
                            <OpenButton
                                held={found.mac.ThisMac ? NO_FIX : elsewhere(on)}
                                label="Sign in"
                                openable={found.mac.ThisMac ? Openable.ofFix(account) : undefined}
                            />
                        )}
                        {loan ? <TakeBackButton gesture={back} shown={shown} vm={on} /> : null}
                    </span>
                }
                trailSize="wide"
            />
            <Withheld fields={[...about.withheld, ...(looksSecret(hint) ? ['hint'] : [])]} />
        </>
    );
}

type ReportProps = { report: AccountsReport; found: Found; go: Go; target: Target | undefined };

function OnThisMac({ report, found, go, target }: ReportProps) {
    const name = found.mac.Name;
    const { machine } = accountsOf(report, name, false);
    const list = machine?.accounts ?? [];
    return (
        <Panel title="Accounts on this Mac">
            {list.length === 0 ? (
                <p className={styles.empty}>
                    {machine
                        ? `No account listed on ${name}.`
                        : sentence([`The answer names no machine ${name}`, report.Engine])}
                </p>
            ) : (
                list.map((account) => (
                    <AccountRow
                        account={account}
                        found={found}
                        go={go}
                        key={account.provider}
                        report={report}
                        target={target}
                    />
                ))
            )}
        </Panel>
    );
}

function SharedWithIt({ report, found, target }: Omit<ReportProps, 'go'>) {
    const name = found.vm?.Name ?? found.mac.Name;
    const { shares } = accountsOf(report, name, true);
    const mac = accountsOf(report, found.mac.Name, false).machine;
    if (shares.length === 0) {
        return null;
    }
    return (
        <Panel title="Shared with it">
            {shares.map((share) => {
                const withheld = looksSecret(share.account);
                const held = (mac?.accounts ?? []).find(
                    (account) => account.provider === share.account,
                );
                return (
                    <ShareRow
                        account={share.account}
                        holds={holdsFor(share.until, report.Now)}
                        key={`${share.vm}/${share.account}`}
                        lead={undefined}
                        sentence={withheld ? '' : sentence(whoOf(held).shown)}
                        shown={withheld ? 'a share whose name is withheld' : share.account}
                        target={target}
                        vm={share.vm}
                    />
                );
            })}
        </Panel>
    );
}

function ShareList({ report, mac }: { report: AccountsReport; mac: string }) {
    const list = report.ShareList ?? [];
    return (
        <Panel note={`from ${mac}`} title="Can be shared">
            {list.length === 0 ? (
                <p className={styles.empty}>None listed.</p>
            ) : (
                list.map((account) => (
                    <Row
                        key={account}
                        lead={<AppIcon name={looksSecret(account) ? '' : account} />}
                        title={looksSecret(account) ? 'a name that is withheld' : account}
                        trail={
                            <span className={styles.trail}>
                                <span className={styles.hint}>Share for</span>
                                {DURATIONS.map((span) => (
                                    <Button disabled key={span}>
                                        {span}
                                    </Button>
                                ))}
                            </span>
                        }
                    />
                ))
            )}
        </Panel>
    );
}

export function AccountsTab({ found, go }: { found: Found; go: Go }) {
    const name = found.vm?.Name ?? found.mac.Name;
    const target = targetOf({ mac: found.mac });
    const accounts = useWatch('accounts.list', { VMs: true }, target);
    if (accounts.kind === 'failed') {
        return (
            <Page>
                <Unanswered error={accounts.error} size="tab" />
            </Page>
        );
    }
    if (accounts.kind === 'reading') {
        return (
            <Page>
                <Reading size="tab" what={`the accounts of ${name}`} />
            </Page>
        );
    }
    const report = accounts.out;
    if (found.vm === undefined) {
        return (
            <Page>
                <NoVerdict line={MAC_LINE} size="tab" />
                <HeldOutcomes>
                    <OnThisMac found={found} go={go} report={report} target={target} />
                </HeldOutcomes>
            </Page>
        );
    }
    return (
        <Page>
            <NoVerdict
                line={`A share is a copy, for a time: when the time ends ${found.mac.Name} takes it back${SEPARATOR}no secret is ever shown here`}
                size="tab"
            />
            <HeldOutcomes>
                <SharedWithIt found={found} report={report} target={target} />
            </HeldOutcomes>
            <ShareList mac={found.mac.Name} report={report} />
            <ChapterDiffers chapter="accounts" found={found} nameOf={(one) => one.Account} />
        </Page>
    );
}
