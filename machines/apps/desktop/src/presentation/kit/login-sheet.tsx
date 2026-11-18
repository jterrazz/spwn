import type { AccountsProgramRow, AccountsStep } from '@jterrazz/station-api';
import { useState } from 'react';
import type { ReactNode } from 'react';

import { codeLeft } from '../../domain/accounts/code-left';
import type { EngineError } from '../../domain/engine-error';
import { aimed } from '../../domain/machines/aim';
import type { Found } from '../../domain/machines/machines';
import { useReread } from '../dependencies/dependencies';
import { useGesture } from '../dependencies/use-gesture';
import { CopyButton } from '../ui/copy-button';
import { AppIcon, Button, Dialog, Screen, StateDot, StatePill } from '../ui/index';
import { detailOf, stepHeadingOf } from '../words/engine-words';
import { engineShown } from '../words/shown';
import { GestureButton, GestureOutcome } from './gestures';
import { PasteCode } from './paste-code';
import { Unanswered } from './readings';
import { useLogin } from './use-login';

import styles from './login-sheet.module.css';

type Program = AccountsProgramRow['Program'];

function Status({ line, left }: { line: string; left: string }) {
    return (
        <p className={styles.status}>
            <span>
                <StateDot {...engineShown('AccountsProgramState', 'waiting')} /> {line}
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

function StepBody({ found, step }: { found: Found; step: AccountsStep }) {
    const site = step.Site ?? '';
    const left = codeLeft(step);
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
                    <Status left={left} line={step.Line} />
                </>
            );
        }
        case 'link': {
            return <Status left={left} line={step.Line} />;
        }
        case 'paste': {
            return (
                <>
                    <div className={styles.well}>
                        <span>{site}</span>
                        <OpenSite found={found} primary={false} step={step} />
                    </div>
                    <PasteCode found={found} step={step} />
                    <Status left={left} line={step.Line} />
                </>
            );
        }
    }
}

function Instruction({ step }: { step: AccountsStep }) {
    switch (step.Kind) {
        case 'device-code': {
            return null;
        }
        case 'link':
        case 'paste': {
            return (
                <p className={styles.instruction}>
                    <b>{stepHeadingOf(step.Kind)}</b>
                    <span>{detailOf('AccountsStepKind', step.Kind)}</span>
                </p>
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

function loginExpired(error: EngineError): boolean {
    return error.wire?.code === 'login.expired';
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
        reread(['accounts.list', 'overview.read']);
        onClose();
    });
    const cancel = <Button onClick={onClose}>Cancel</Button>;
    const sheet = (body: ReactNode, actions: ReactNode, step?: AccountsStep) => (
        <Screen>
            <Dialog
                actions={actions}
                lead={<AppIcon name={program} />}
                onClose={onClose}
                title={step === undefined ? title : step.Title}
            >
                {body}
            </Dialog>
        </Screen>
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
            step,
        );
    }
    const { step } = seen;
    return sheet(
        <div className={styles.body}>
            <Instruction step={step} />
            <StepBody found={found} step={step} />
        </div>,
        <>
            {cancel}
            <StepOpens found={found} step={step} />
        </>,
        step,
    );
}

type SheetProps = {
    found: Found;
    program: Program;
    title: string;
    onClose: () => void;
};

export function LoginSheet({ found, program, title, onClose }: SheetProps) {
    const [round, setRound] = useState(0);
    return (
        <Round
            found={found}
            key={round}
            onClose={onClose}
            onNewCode={() => {
                setRound(round + 1);
            }}
            program={program}
            title={title}
        />
    );
}
