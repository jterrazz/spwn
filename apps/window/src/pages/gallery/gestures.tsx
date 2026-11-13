import { useState } from 'react';

import { Button, ConfirmDialog, Panel, ProgressLine, Row, StatePill } from '../../design/index';
import { EngineError } from '../../engine/engine-error';
import { Unknown } from '../kit/gestures';
import { Unanswered } from '../kit/readings';
import { nothing, Section } from './section';

import styles from './gallery.module.css';

const STOPPED = new EngineError({
    hint: 'station engine start',
    message: 'the engine is stopped on This Mac',
});

const DROPPED = new EngineError(undefined, 'the engine closed the connection', 'lost');

type Asked = 'name' | 'none' | 'yes';

export function Confirms() {
    const [asked, setAsked] = useState<Asked>('none');
    const close = () => {
        setAsked('none');
    };
    return (
        <Section title="Confirm dialog">
            <Button
                onClick={() => {
                    setAsked('yes');
                }}
            >
                Ask a yes
            </Button>
            <Button
                onClick={() => {
                    setAsked('name');
                }}
            >
                Ask the name typed
            </Button>
            {asked === 'yes' ? (
                <ConfirmDialog
                    onCancel={close}
                    onConfirm={close}
                    title="Stop linux-1?"
                    verb="Stop"
                />
            ) : null}
            {asked === 'name' ? (
                <ConfirmDialog
                    onCancel={close}
                    onConfirm={close}
                    said="deleting a server needs its name"
                    title="Delete infra?"
                    typed="infra"
                    verb="Delete infra"
                />
            ) : null}
        </Section>
    );
}

export function GestureStates() {
    return (
        <Section title="Gesture states">
            <div className={styles.stack}>
                <div className={styles.samples}>
                    <Button busy>Stop</Button>
                    <ProgressLine task="Updating the mise apps" />
                    <ProgressLine detail="brew exited 1" failed task="Updating the brew apps" />
                    <StatePill tone="ok" word="done" />
                </div>
                <Panel title="Needs you">
                    <Row
                        lead={<StatePill tone="warn" word="Needs a look" />}
                        sentence="jterrazz-macbook-pro · Apps"
                        title="2 brew updates waiting"
                        trail={
                            <>
                                <ProgressLine task="Updating the brew apps" />
                                <Button busy>apps.update</Button>
                            </>
                        }
                    />
                    <Row
                        lead={<StatePill tone="ok" word="settled" />}
                        sentence="jterrazz-macbook-pro · Apps"
                        title="1 mise update waiting"
                    />
                </Panel>
                <div className={styles.samples}>
                    <Unanswered error={STOPPED} size="panel" />
                    <Button onClick={nothing} variant="link">
                        Dismiss
                    </Button>
                </div>
                <div className={styles.samples}>
                    <Unknown error={DROPPED} onDismiss={nothing} />
                </div>
            </div>
        </Section>
    );
}
