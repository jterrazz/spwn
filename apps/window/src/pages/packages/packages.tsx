import type { CallName, MachinefileDrift, Target } from '@jterrazz/station-api';
import { useState } from 'react';

import { Screen } from '../../design/index';
import type { Reading as Read } from '../../engine/react';
import { useRead } from '../../engine/react';
import { useGesture } from '../../engine/use-gesture';
import { useRouter } from '../../shell/router';
import { GestureButton, GestureOutcome } from '../kit/gestures';
import { NotGiven, NoVerdict, Reading } from '../kit/readings';
import { Page } from '../page';
import { Differs } from './differs';
import { Inventory } from './inventory';
import { PlanView } from './plan';

import styles from './packages.module.css';

function FileLink({ file }: { file: string }) {
    const { go } = useRouter();
    return (
        <>
            Its machine file is{' '}
            <button
                className={styles.fileLink}
                onClick={() => {
                    go({ file, page: 'machine-files' });
                }}
                type="button"
            >
                {file}
            </button>
        </>
    );
}

function FileState({ drift, name }: { drift: Read<MachinefileDrift>; name: string }) {
    if (drift.kind === 'reading') {
        return <Reading size="tab" what={`the file ${name} follows`} />;
    }
    return (
        <NoVerdict
            line={
                drift.kind === 'read' ? (
                    <FileLink file={drift.out.Plan.Name} />
                ) : (
                    'What the engine answered is below'
                )
            }
            size="tab"
        />
    );
}

const UPGRADE_REREADS: readonly CallName[] = ['machinefile.drift', 'machinefile.plan'];

type Aimed = { target: Target | undefined; name: string };

function Upgrades({ target, name }: Aimed) {
    const upgrade = useGesture('machinefile.upgrade', {
        input: {},
        rereads: UPGRADE_REREADS,
        subject: name,
        target,
    });
    return (
        <>
            <NotGiven
                action={
                    <GestureButton
                        gesture={upgrade}
                        label="Upgrade now"
                        title={`Upgrade what the file of ${name} lists?`}
                        verb="Upgrade"
                    />
                }
                title="Upgrades"
                what="When it upgrades by itself, and its last run"
            />
            <GestureOutcome gesture={upgrade} />
        </>
    );
}

function Setup({ target, name, onPlan }: Aimed & { onPlan: () => void }) {
    const drift = useRead('machinefile.drift', {}, target);
    return (
        <Page>
            <FileState drift={drift} name={name} />
            {drift.kind === 'read' ? <Differs onPlan={onPlan} plan={drift.out.Plan} /> : null}
            <Upgrades name={name} target={target} />
            <Inventory drift={drift} target={target} />
        </Page>
    );
}

export function PackagesTab({ name, target }: Aimed) {
    const [planning, setPlanning] = useState(false);
    return (
        <Screen>
            {planning ? (
                <PlanView
                    back={() => {
                        setPlanning(false);
                    }}
                    name={name}
                    target={target}
                />
            ) : (
                <Setup
                    name={name}
                    onPlan={() => {
                        setPlanning(true);
                    }}
                    target={target}
                />
            )}
        </Screen>
    );
}
