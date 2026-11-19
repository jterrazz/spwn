import type { UpdateFlow, UpdateState } from '../../../application/use-cases/update-flow';
import { START_LINE } from '../../../domain/core';
import { Button, StatePill } from '../../ui/index';
import { shownOrWithheld } from '../../ui/secret';
import { updateState } from '../../words/shown';
import { Acts, Head, Output, Progress, Status } from './core-parts';

import styles from './core.module.css';

const INSTALLER = "The installer's output";
const STARTED = `What ${START_LINE} said`;

type Shown = Exclude<UpdateState, { kind: 'idle' }>;

function failedLead(state: Extract<UpdateState, { kind: 'failed' }>): string {
    const what = state.step === 'install' ? 'The installer' : START_LINE;
    return state.code === undefined
        ? `The window could not run ${state.step === 'install' ? 'the installer' : START_LINE}.`
        : `${what} ended with exit code ${String(state.code)}. Its output is kept below.`;
}

function headOf(state: Shown): { title: string; lead: string } {
    switch (state.kind) {
        case 'updating': {
            return {
                lead: 'The installer runs first, then the engine restarts once.',
                title: `Updating to ${state.to}.`,
            };
        }
        case 'restarting': {
            return {
                lead: `The installer is done. ${START_LINE} restarts the engine.`,
                title: `Updating to ${state.to}.`,
            };
        }
        case 'updated': {
            return {
                lead: 'The installer is done and the engine restarted. The head of this page is what it says now.',
                title: 'Updated.',
            };
        }
        case 'failed': {
            const unsaid =
                state.step === 'install'
                    ? 'The update did not finish.'
                    : 'The engine did not restart.';
            return {
                lead: failedLead(state),
                title: state.said === '' ? unsaid : shownOrWithheld(state.said),
            };
        }
    }
}

function Running({ flow, state }: { flow: UpdateFlow; state: Shown }) {
    if (state.kind === 'updating') {
        return (
            <>
                <Status aside="· its own output below" said="Running the installer" />
                <Progress label="Updating spwn" />
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
    if (state.kind === 'restarting') {
        return (
            <>
                <Status said="Restarting the engine" />
                <Progress label="Restarting the engine" />
            </>
        );
    }
    return (
        <Acts>
            {state.kind === 'failed' ? (
                <Button
                    onClick={() => {
                        flow.retry();
                    }}
                    variant="primary"
                >
                    Retry
                </Button>
            ) : null}
            <Button
                onClick={() => {
                    flow.done();
                }}
            >
                Done
            </Button>
        </Acts>
    );
}

export function UpdatePanel({
    flow,
    state,
    wide = false,
}: {
    flow: UpdateFlow;
    state: Shown;
    wide?: boolean;
}) {
    const { title, lead } = headOf(state);
    const { install, start } = state.lines;
    return (
        <section
            aria-label="Update"
            className={wide ? `${styles.panel} ${styles.wide}` : styles.panel}
        >
            <span className={styles.pill}>
                <StatePill {...updateState(state.kind)} />
            </span>
            <Head title={title}>{lead}</Head>
            {install.length > 0 ? <Output label={INSTALLER} lines={install} /> : null}
            {start.length > 0 ? <Output label={STARTED} lines={start} /> : null}
            <Running flow={flow} state={state} />
        </section>
    );
}
