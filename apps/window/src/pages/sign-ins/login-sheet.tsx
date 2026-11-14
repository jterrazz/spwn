import type { AccountsProgramRow, AccountsStep } from '@jterrazz/station-api';
import { useState } from 'react';
import type { ReactNode } from 'react';

import { CopyButton } from '../../design/copy-button';
import { AppIcon, Button, Dialog, StateDot, StatePill } from '../../design/index';
import type { EngineError } from '../../engine/engine-error';
import { wordOf } from '../../engine/engine-words';
import { loginExpired } from '../../engine/provisional';
import { useReread } from '../../engine/react';
import { engineShown } from '../../engine/shown';
import { useGesture } from '../../engine/use-gesture';
import type { Found } from '../../model/machines';
import { GestureButton, GestureOutcome } from '../kit/gestures';
import { Unanswered } from '../kit/readings';
import { aimed } from './aim';
import { PasteCode } from './paste-code';
import { codeLeft } from './sign-ins-of';
import { useLogin } from './use-login';

import styles from './sign-ins.module.css';

type Program = AccountsProgramRow['Program'];

function Status({ word, left }: { word: string; left: string }) {
    return (
        <p className={styles.status}>
            <span>
                <StateDot {...engineShown('AccountsProgramState', 'waiting')} /> {word}
            </span>
            <span>
                expires in <span className={styles.left}>{left}</span>
            </span>
        </p>
    );
}

function OpenSite({
    found,
    step,
    primary,
}: {
    found: Found;
    step: AccountsStep;
    primary: boolean;
}) {
    const site = step.Site ?? '';
    const open = useGesture('accounts.open', {
        ...aimed(found, { Exchange: step.Exchange }),
        rereads: [],
        subject: site,
    });
    return (
        <>
            <GestureButton
                gesture={open}
                label={`Open ${site}`}
                title={`Open ${site}?`}
                variant={primary ? 'primary' : 'plain'}
            />
            {open.state.kind === 'failed' ? <GestureOutcome gesture={open} /> : null}
        </>
    );
}

function StepBody({ found, step, machine }: { found: Found; step: AccountsStep; machine: string }) {
    const site = step.Site ?? '';
    const left = codeLeft(step);
    const waiting = wordOf('AccountsProgramState', 'waiting').Word;
    switch (step.Kind) {
        case 'device-code': {
            return (
                <>
                    <div className={styles.well}>
                        <span className={styles.code}>{step.Code ?? ''}</span>
                        <CopyButton
                            label="Copy code"
                            name="Copy code"
                            text={step.Code ?? ''}
                            variant="plain"
                        />
                    </div>
                    <Status left={left} word={`${waiting}: ${site}`} />
                </>
            );
        }
        case 'link': {
            return <Status left={left} word={`${waiting}: ${site}`} />;
        }
        case 'paste': {
            return (
                <>
                    <div className={styles.well}>
                        <span>{site}</span>
                        <OpenSite found={found} primary={false} step={step} />
                    </div>
                    <PasteCode found={found} step={step} />
                    <Status left={left} word={`${waiting}: ${machine}`} />
                </>
            );
        }
    }
}

function StepOpens({ found, step }: { found: Found; step: AccountsStep }) {
    switch (step.Kind) {
        case 'device-code':
        case 'link': {
            return <OpenSite found={found} primary step={step} />;
        }
        case 'paste': {
            return null;
        }
    }
}

function Expired({ step, error }: { step: AccountsStep; error: EngineError }) {
    return (
        <div className={styles.body}>
            <div className={styles.well}>
                <span className={`${styles.code} ${styles.struck}`}>
                    {step.Code ?? step.Site ?? ''}
                </span>
                <StatePill {...engineShown('CoreFailCode', 'login.expired')} />
            </div>
            <p className={styles.said}>{error.message}</p>
        </div>
    );
}

type RoundProps = {
    found: Found;
    program: Program;
    title: string;
    onClose: () => void;
    onNewCode: () => void;
};

function Round({ found, program, title, onClose, onNewCode }: RoundProps) {
    const reread = useReread();
    const seen = useLogin(found, program, () => {
        reread(['accounts.list']);
        onClose();
    });
    const cancel = <Button onClick={onClose}>Cancel</Button>;
    const sheet = (body: ReactNode, actions: ReactNode, consequence?: string) => (
        <Dialog
            actions={actions}
            consequence={consequence}
            lead={<AppIcon name={program} />}
            onClose={onClose}
            title={title}
        >
            {body}
        </Dialog>
    );
    if (seen.kind === 'asking') {
        return sheet(<AppIcon name={program} />, cancel);
    }
    if (seen.kind === 'failed') {
        const { error, step } = seen;
        if (step === undefined || !loginExpired(error)) {
            return sheet(<Unanswered error={error} size="panel" />, cancel);
        }
        const again = (
            <Button onClick={onNewCode} variant="primary">
                New code
            </Button>
        );
        return sheet(
            <Expired error={error} step={step} />,
            <>
                {cancel}
                {again}
            </>,
            step.Expired,
        );
    }
    const { step } = seen;
    const machine = found.vm?.Name ?? found.mac.Name;
    return sheet(
        <div className={styles.body}>
            <StepBody found={found} machine={machine} step={step} />
        </div>,
        <>
            {cancel}
            <StepOpens found={found} step={step} />
        </>,
        step.Line,
    );
}

type SheetProps = {
    found: Found;
    program: Program;
    action: AccountsProgramRow['Action'];
    onClose: () => void;
};

export function LoginSheet({ found, program, action, onClose }: SheetProps) {
    const [round, setRound] = useState(0);
    const machine = found.vm?.Name ?? found.mac.Name;
    return (
        <Round
            found={found}
            key={round}
            onClose={onClose}
            onNewCode={() => {
                setRound(round + 1);
            }}
            program={program}
            title={`${wordOf('AccountsProgramAction', action).Word} to ${program} on ${machine}`}
        />
    );
}
