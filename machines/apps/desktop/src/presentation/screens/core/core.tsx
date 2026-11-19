import type { CoreFlow, CoreState } from '../../../application/use-cases/core-flow';
import { INSTALL_FOLDER, START_LINE } from '../../../domain/core';
import { useCore, useUpdate } from '../../dependencies/dependencies';
import { Button } from '../../ui/index';
import { shownOrWithheld } from '../../ui/secret';
import {
    Acts,
    Assurances,
    Command,
    Fine,
    Head,
    Mark,
    Mono,
    Output,
    Pill,
    Progress,
    Rows,
    Status,
} from './core-parts';
import { ON_DISK, PROMISES, PUTS, STARTING } from './core-rows';
import { UpdatePanel } from './update';

import styles from './core.module.css';

const OUTPUT = "The installer's output";

function Offer({ flow }: { flow: CoreFlow }) {
    return (
        <>
            <Mark />
            <Head title="spwn is not installed on this Mac.">
                This window shows the machines the spwn engine runs. Install it here, then start the
                engine.
            </Head>
            <Rows label="What it puts on this Mac" rows={PUTS} />
            <Assurances said={PROMISES} />
            <Command />
            <Acts>
                <Button
                    onClick={() => {
                        flow.install();
                    }}
                    variant="primary"
                >
                    Install
                </Button>
                <Button
                    onClick={() => {
                        flow.waitForTerminal();
                    }}
                    variant="link"
                >
                    I&apos;ll install it from the terminal
                </Button>
            </Acts>
        </>
    );
}

function Installing({
    flow,
    state,
}: {
    flow: CoreFlow;
    state: Extract<CoreState, { kind: 'installing' }>;
}) {
    return (
        <>
            <Mark />
            <Head title="Installing spwn.">
                The installer runs the command below. Nothing lands in <Mono>{INSTALL_FOLDER}</Mono>{' '}
                until every checksum matches.
            </Head>
            <Command />
            <Status aside="· its own output below" said="Running the installer" />
            <Progress label="Installing spwn" />
            <Output label={OUTPUT} lines={state.lines} />
            <Acts>
                <Button
                    onClick={() => {
                        flow.cancel();
                    }}
                >
                    Cancel
                </Button>
            </Acts>
        </>
    );
}

function Waiting({ flow }: { flow: CoreFlow }) {
    return (
        <>
            <Head title="Waiting for spwn on this Mac.">
                Run the command in a terminal. This page moves on by itself when{' '}
                <Mono>station</Mono> appears.
            </Head>
            <Command />
            <Status aside="· every few seconds" said="Looking for station" />
            <Acts>
                <Button
                    onClick={() => {
                        flow.installHere();
                    }}
                >
                    Install it here instead
                </Button>
            </Acts>
        </>
    );
}

type Starts = Extract<CoreState, { kind: 'installed' | 'started' | 'starting' }>;

function Installed({ flow, state }: { flow: CoreFlow; state: Starts }) {
    const lines = state.kind === 'installed' ? [] : state.lines;
    return (
        <>
            <Pill kind="installed" />
            <Head title="spwn is installed. Start the engine next.">
                The engine runs your machines. The installer does not start it, so nothing runs yet.
            </Head>
            <Rows label="On this Mac" rows={ON_DISK} />
            <Rows label="Starting the engine" rows={STARTING} />
            {state.kind === 'started' ? <Status said="Waiting for the engine to answer" /> : null}
            {lines.length > 0 ? <Output label={`What ${START_LINE} said`} lines={lines} /> : null}
            <Acts>
                <Button
                    busy={state.kind !== 'installed'}
                    onClick={() => {
                        flow.start();
                    }}
                    variant="primary"
                >
                    Start the engine
                </Button>
                {state.kind === 'installed' ? (
                    <Button
                        onClick={() => {
                            flow.later();
                        }}
                        variant="link"
                    >
                        Later
                    </Button>
                ) : null}
                <span className={styles.grow} />
                <Fine>
                    <Mono>{START_LINE}</Mono>
                </Fine>
            </Acts>
        </>
    );
}

function Later({ flow }: { flow: CoreFlow }) {
    return (
        <>
            <Pill kind="installed" />
            <Head title="The engine is not started.">
                spwn is on this Mac. Start the engine when you want to see your machines.
            </Head>
            <Acts>
                <Button
                    onClick={() => {
                        flow.start();
                    }}
                    variant="primary"
                >
                    Start the engine
                </Button>
            </Acts>
        </>
    );
}

function failedLead(state: Extract<CoreState, { kind: 'failed' }>): string {
    const what = state.of === 'install' ? 'The installer' : START_LINE;
    return state.code === undefined
        ? `The window could not run ${state.of === 'install' ? 'the installer' : START_LINE}.`
        : `${what} ended with exit code ${String(state.code)}. Its output is kept below.`;
}

function Failed({
    flow,
    state,
}: {
    flow: CoreFlow;
    state: Extract<CoreState, { kind: 'failed' }>;
}) {
    const install = state.of === 'install';
    const unsaid = install ? 'The install did not finish.' : 'The engine did not start.';
    const title = state.said === '' ? unsaid : shownOrWithheld(state.said);
    return (
        <>
            <Pill kind="failed" />
            <Head title={title}>{failedLead(state)}</Head>
            {state.lines.length > 0 ? (
                <Output label={install ? OUTPUT : `What ${START_LINE} said`} lines={state.lines} />
            ) : null}
            {install ? <Command /> : null}
            <Acts>
                <Button
                    onClick={() => {
                        flow.retry();
                    }}
                    variant="primary"
                >
                    Retry
                </Button>
                {install ? (
                    <Button
                        onClick={() => {
                            flow.waitForTerminal();
                        }}
                        variant="link"
                    >
                        I&apos;ll install it from the terminal
                    </Button>
                ) : null}
            </Acts>
        </>
    );
}

function Shown({ flow, state }: { flow: CoreFlow; state: CoreState }) {
    switch (state.kind) {
        case 'looking': {
            return null;
        }
        case 'missing': {
            return <Offer flow={flow} />;
        }
        case 'installing': {
            return <Installing flow={flow} state={state} />;
        }
        case 'waiting': {
            return <Waiting flow={flow} />;
        }
        case 'installed': {
            return state.later ? <Later flow={flow} /> : <Installed flow={flow} state={state} />;
        }
        case 'starting':
        case 'started': {
            return <Installed flow={flow} state={state} />;
        }
        case 'failed': {
            return <Failed flow={flow} state={state} />;
        }
    }
}

export function CorePage() {
    const { flow, state } = useCore();
    const update = useUpdate();
    if (update.state.kind !== 'idle') {
        return (
            <div className={styles.center}>
                <UpdatePanel flow={update.flow} state={update.state} />
            </div>
        );
    }
    return (
        <div className={styles.center}>
            {state.kind === 'looking' ? null : (
                <section aria-label="spwn on this Mac" className={styles.panel}>
                    <Shown flow={flow} state={state} />
                </section>
            )}
        </div>
    );
}
