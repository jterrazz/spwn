import type { CallName, MachinefilePlan, Target } from '@jterrazz/station-api';

import { AppIcon, Button, Facts, Panel, Row, Tile, Value } from '../../design/index';
import type { PlanLine } from '../../engine/provisional';
import { useRead } from '../../engine/react';
import type { Reading as Read } from '../../engine/react';
import { CONDITION_TONES } from '../../engine/tones';
import { useGesture } from '../../engine/use-gesture';
import { ACTION_WORDS, CONDITION_WORDS, NEED_WORDS, RESTART_WORDS } from '../../engine/words';
import { sentence } from '../../model/format';
import { GestureButton, GestureOutcome } from '../kit/gestures';
import { Reading, Unanswered } from '../kit/readings';
import { Page, Tiles } from '../page';
import { byAction } from './model';

import styles from './packages.module.css';

function Steps({ lines, title }: { lines: readonly PlanLine[]; title: string }) {
    return (
        <Panel title={title}>
            {lines.map((line) => (
                <Row
                    key={`${line.Step.Kind}/${line.Step.Package}`}
                    lead={<AppIcon kind={line.Step.Kind} name={line.Step.Package} />}
                    sentence={line.Step.Why === '' ? line.Seen.Detail : line.Step.Why}
                    size="slim"
                    state={{
                        tone: CONDITION_TONES[line.Seen.Condition],
                        word: CONDITION_WORDS[line.Seen.Condition],
                    }}
                    title={line.Step.Package}
                    trail={line.Step.Origin.Name === '' ? null : `from ${line.Step.Origin.Name}`}
                    value={line.Step.Version === '' ? null : <Value>{line.Step.Version}</Value>}
                />
            ))}
        </Panel>
    );
}

export function Asks({ plan, title }: { plan: MachinefilePlan; title: string }) {
    const needs = plan.Needs ?? [];
    if (needs.length === 0) {
        return null;
    }
    return (
        <Panel title={title}>
            {needs.map((need) => (
                <Row
                    key={need.Need}
                    sentence={sentence(need.Packages ?? [])}
                    size="slim"
                    title={NEED_WORDS[need.Need]}
                />
            ))}
        </Panel>
    );
}

function Afterwards({ plan }: { plan: MachinefilePlan }) {
    const skipped = plan.Skipped ?? [];
    const restarts = plan.Restarts ?? [];
    return (
        <>
            {skipped.length === 0 ? null : (
                <Panel title="Skipped">
                    {skipped.map((skip) => (
                        <Row
                            key={skip.Package}
                            sentence={skip.Why}
                            size="slim"
                            title={skip.Package}
                        />
                    ))}
                </Panel>
            )}
            {restarts.length === 0 ? null : (
                <Panel title="Once applied, it needs">
                    {restarts.map((restart) => (
                        <Row key={restart} size="slim" title={RESTART_WORDS[restart]} />
                    ))}
                </Panel>
            )}
        </>
    );
}

export function PlanSteps({ plan }: { plan: MachinefilePlan }) {
    return (
        <>
            <Panel padded title="Its machine file">
                <Facts facts={[{ label: 'Its file', value: plan.Name }]} />
            </Panel>
            <Asks plan={plan} title="It asks" />
            {byAction(plan.Lines ?? []).map(([action, lines]) => (
                <Steps key={action} lines={lines} title={`To ${ACTION_WORDS[action]}`} />
            ))}
            {byAction(plan.Extra ?? []).map(([action, lines]) => (
                <Steps
                    key={`extra/${action}`}
                    lines={lines}
                    title={`Extra, to ${ACTION_WORDS[action]}`}
                />
            ))}
            <Afterwards plan={plan} />
        </>
    );
}

const APPLY_REREADS: readonly CallName[] = ['machinefile.drift', 'machinefile.plan'];

type PlanProps = { target: Target | undefined; name: string; back: () => void };

function PlanBar({ target, name, back }: PlanProps) {
    const apply = useGesture('machinefile.apply', {
        input: {},
        rereads: APPLY_REREADS,
        subject: name,
        target,
    });
    return (
        <div className={styles.foot}>
            <span className={styles.item}>
                <GestureOutcome gesture={apply} />
            </span>
            <span className={styles.item}>
                <Button onClick={back}>Back</Button>
                <GestureButton
                    gesture={apply}
                    label="Apply"
                    title={`Apply the plan of ${name}?`}
                    variant="primary"
                />
            </span>
        </div>
    );
}

function PlanBody({ plan, name }: { plan: Read<MachinefilePlan>; name: string }) {
    if (plan.kind === 'failed') {
        return <Unanswered error={plan.error} size="tab" />;
    }
    if (plan.kind === 'reading') {
        return <Reading size="tab" what={`the plan of ${name}`} />;
    }
    return (
        <>
            <div className={styles.planHead}>
                <h1 className={styles.planTitle}>Plan for {name}</h1>
                <p className={styles.planLine}>
                    What applying {plan.out.Name} would change. Nothing has changed yet.
                </p>
            </div>
            <PlanCounts plan={plan.out} />
            <PlanSteps plan={plan.out} />
        </>
    );
}

function PlanCounts({ plan }: { plan: MachinefilePlan }) {
    const groups = byAction(plan.Lines ?? []);
    if (groups.length === 0) {
        return null;
    }
    return (
        <Tiles>
            {groups.map(([action, lines]) => (
                <Tile
                    foot=""
                    key={action}
                    label={`To ${ACTION_WORDS[action]}`}
                    value={lines.length}
                />
            ))}
        </Tiles>
    );
}

export function PlanView(props: PlanProps) {
    const { target, name } = props;
    const plan = useRead('machinefile.plan', {}, target);
    return (
        <Page>
            <PlanBody name={name} plan={plan} />
            <PlanBar {...props} />
        </Page>
    );
}
