import styles from './guide.module.css';

// value: where a step of an act stands: done, the one under way (now), the one waiting on its owner, the one that
// value: failed, and the ones still to come (next).
export type GuideStepStatus = 'done' | 'failed' | 'next' | 'now' | 'waiting';

export type GuideStep = {
    id: string;
    label: string;
    status: GuideStepStatus;
    detail?: string | undefined;
};

// value: a rail holds seven lines at most before its done steps fold into one.
const RAIL_ROOM = 7;

const SAID: Readonly<Record<GuideStepStatus, string>> = {
    done: ', done',
    failed: ', failed',
    next: '',
    now: '',
    waiting: ', waits for you',
};

export function isCurrent(step: GuideStep): boolean {
    return step.status === 'now' || step.status === 'waiting' || step.status === 'failed';
}

function Tick() {
    return (
        <svg aria-hidden="true" className={styles.tick} viewBox="0 0 10 10">
            <polyline points="2,5.2 4.2,7.3 8,2.8" />
        </svg>
    );
}

function Step({ step }: { step: GuideStep }) {
    const current = isCurrent(step);
    return (
        <li
            aria-current={current ? 'step' : undefined}
            className={styles.step}
            data-status={step.status}
        >
            <span aria-hidden="true" className={styles.knob}>
                {step.status === 'done' ? <Tick /> : null}
                {step.status === 'failed' ? '!' : null}
            </span>
            <span className={styles.label}>
                {step.label}
                <span className={styles.said}>{SAID[step.status]}</span>
                {current && (step.detail ?? '') !== '' ? (
                    <span className={styles.detail}>{step.detail}</span>
                ) : null}
            </span>
        </li>
    );
}

function Folded({ count }: { count: number }) {
    return (
        <li className={`${styles.step} ${styles.fold}`}>
            <span aria-hidden="true" className={styles.knob}>
                {count}
            </span>
            <span className={styles.label}>{`${String(count)} steps done`}</span>
        </li>
    );
}

function placeOf(steps: readonly GuideStep[]): number {
    const at = steps.findIndex((step) => isCurrent(step));
    return at === -1 ? steps.filter((step) => step.status === 'done').length : at + 1;
}

export type GuideRailProps = {
    act: string;
    subject: string;
    actId: string;
    steps: readonly GuideStep[];
    ended?: string | undefined;
};

export function GuideRail({ act, subject, actId, steps, ended }: GuideRailProps) {
    const done = steps.filter((step) => step.status === 'done');
    const folds = steps.length > RAIL_ROOM && done.length > 1;
    const shown = folds ? steps.filter((step) => step.status !== 'done') : steps;
    return (
        <div className={styles.rail}>
            <h2 className={styles.act} id={actId}>
                <span className={styles.actName}>{act}</span> <span>{subject}</span>
            </h2>
            <ol aria-label="Steps" className={styles.steps}>
                {folds ? <Folded count={done.length} /> : null}
                {shown.map((step) => (
                    <Step key={step.id} step={step} />
                ))}
            </ol>
            <p className={styles.foot}>
                {ended ?? `${String(placeOf(steps))} of ${String(steps.length)}`}
            </p>
        </div>
    );
}
