import { useId, useLayoutEffect, useRef } from 'react';

import { SEPARATOR } from '../../domain/separator';
import { Button, Screen } from './button';
import { Modal } from './dialog';
import { Fingerprint, HideMark, KeptMark } from './guide-marks';
import { GuideRail, isCurrent } from './guide-rail';
import type { GuideStep } from './guide-rail';
import { Illustration } from './illustrations/illustration';
import type { IllustrationName } from './illustrations/illustration';
import type { Measured } from './measure';

import styles from './guide.module.css';

export type { GuideStep, GuideStepStatus } from './guide-rail';

export type GuideAction = { label: string; onPress: () => void };

// value: how far a step is: a measure when the engine has one (a bar and its figure), else its last live line.
export type GuideProgress =
    | { kind: 'live'; line: string }
    | { kind: 'measure'; done: number; total: number; unit: string; eta?: string | undefined };

// value: the sentence macOS will show, quoted before the owner's press opens its prompt.
export type GuidePrompt = { lead: string; quote: string };

// value: the current step's view, all of it the engine's: what is happening and what to do, the one sentence, the
// value: drawing, how far it is, what a failure kept, and whose move it is, which alone makes the action primary.
export type GuideView = {
    illustration: IllustrationName;
    headline: { now: string; next: string };
    sentence: string;
    move: 'engine' | 'owner';
    progress?: GuideProgress | undefined;
    prompt?: GuidePrompt | undefined;
    kept?: string | undefined;
    action?: GuideAction | undefined;
    quiet?: GuideAction | undefined;
};

export type GuideProps = {
    act: string;
    subject: string;
    steps: readonly GuideStep[];
    view: GuideView;
    ended?: string | undefined;
    onHide?: (() => void) | undefined;
};

const stay = (): void => undefined;

function Measure({
    progress,
    label,
}: {
    progress: GuideProgress & { kind: 'measure' };
    label: string;
}) {
    const { done, total, unit, eta } = progress;
    const figure = [`${String(done)} of ${String(total)} ${unit}`, eta ?? '']
        .filter((part) => part !== '')
        .join(SEPARATOR);
    const share = total > 0 ? Math.max(0, Math.min(100, (done / total) * 100)) : 0;
    const fill: Measured = { '--fill': `${share.toFixed(1)}%` };
    return (
        <div className={styles.measure}>
            <div
                aria-label={label}
                aria-valuemax={total}
                aria-valuemin={0}
                aria-valuenow={done}
                aria-valuetext={figure}
                className={styles.track}
                role="progressbar"
            >
                <i className={styles.fill} style={fill} />
            </div>
            <p aria-hidden="true" className={styles.figure}>
                {figure}
            </p>
        </div>
    );
}

function Progress({ progress, label }: { progress: GuideProgress | undefined; label: string }) {
    if (progress === undefined) {
        return null;
    }
    if (progress.kind === 'measure') {
        return <Measure label={label} progress={progress} />;
    }
    return (
        <p className={styles.live}>
            <i aria-hidden="true" className={styles.pulse} />
            {progress.line}
        </p>
    );
}

function Notes({ prompt, kept }: Pick<GuideView, 'kept' | 'prompt'>) {
    return (
        <>
            {prompt === undefined ? null : (
                <p className={styles.note}>
                    <Fingerprint />
                    <span>
                        {prompt.lead} <b>{`“${prompt.quote}”`}</b>
                    </span>
                </p>
            )}
            {(kept ?? '') === '' ? null : (
                <p className={styles.note}>
                    <KeptMark />
                    <span>{kept}</span>
                </p>
            )}
        </>
    );
}

function Actions({ view }: { view: GuideView }) {
    const { action, quiet } = view;
    return (
        <div className={styles.actions}>
            {action === undefined ? null : (
                <Button
                    onClick={action.onPress}
                    variant={view.move === 'owner' ? 'primary' : 'plain'}
                >
                    {action.label}
                </Button>
            )}
            {quiet === undefined ? null : (
                <button className={styles.quiet} onClick={quiet.onPress} type="button">
                    {quiet.label}
                </button>
            )}
        </div>
    );
}

// invariant: a view that drops the control holding the keyboard hands it to the dialog, never to the page under it.
function useKeyboardKept() {
    const sheet = useRef<HTMLDivElement>(null);
    useLayoutEffect(() => {
        const dialog = sheet.current?.closest<HTMLElement>('[role="dialog"]');
        const lost = document.activeElement === null || document.activeElement === document.body;
        if (dialog && lost) {
            dialog.focus({ preventScroll: true });
        }
    });
    return sheet;
}

export function Guide({ act, subject, steps, view, ended, onHide }: GuideProps) {
    const actId = useId();
    const sheet = useKeyboardKept();
    const label = steps.find((step) => isCurrent(step))?.label ?? act;
    const said = `${view.headline.now} ${view.headline.next}`;
    return (
        <Modal labelledBy={actId} onClose={onHide ?? stay} opensOn="dialog" place="centre">
            <Screen>
                <div className={styles.sheet} ref={sheet}>
                    <GuideRail
                        act={act}
                        actId={actId}
                        ended={ended}
                        steps={steps}
                        subject={subject}
                    />
                    <div className={styles.body}>
                        {onHide === undefined ? null : (
                            <button className={styles.hide} onClick={onHide} type="button">
                                <HideMark />
                                Hide
                            </button>
                        )}
                        <div className={styles.art}>
                            <Illustration name={view.illustration} />
                        </div>
                        <h3 className={styles.headline}>
                            {view.headline.now}{' '}
                            <span className={styles.next}>{view.headline.next}</span>
                        </h3>
                        <p className={styles.sentence}>{view.sentence}</p>
                        <Progress label={label} progress={view.progress} />
                        <Notes kept={view.kept} prompt={view.prompt} />
                        <Actions view={view} />
                        <p aria-label={said} className={styles.said} role="status">
                            {said}
                        </p>
                    </div>
                </div>
            </Screen>
        </Modal>
    );
}
