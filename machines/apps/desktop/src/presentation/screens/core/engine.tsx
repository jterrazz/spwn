import type { MachinesRelease } from '@jterrazz/station-api';

import { INSTALL_LINE } from '../../../domain/core';
import { SEPARATOR } from '../../../domain/separator';
import { dayAndClock } from '../../../domain/when';
import { useHello, useOffers, useRead, useUpdate } from '../../dependencies/dependencies';
import { useGesture } from '../../dependencies/use-gesture';
import { GestureButton, GestureOutcome } from '../../kit/gestures';
import { Openable } from '../../kit/openable';
import { OpenButton } from '../../kit/opens';
import { NoVerdict, Reading, Unanswered, VerdictHead } from '../../kit/readings';
import { Button, Panel, Row } from '../../ui/index';
import { shownOrWithheld } from '../../ui/secret';
import { RELEASE_WORDS } from '../../words/words';
import { Fine, Mono } from './core-parts';
import { UpdatePanel } from './update';

import page from '../../ui/page.module.css';
import styles from './core.module.css';

const NO_NOTES = 'The engine names no release notes';

function checkedOf(release: MachinesRelease): string | undefined {
    const at = dayAndClock(release.Checked);
    return at === undefined ? undefined : `${RELEASE_WORDS.checked} ${at}`;
}

function refusedOf(release: MachinesRelease): string | undefined {
    const { Refused } = release;
    if (Refused === undefined) {
        return undefined;
    }
    const at = dayAndClock(Refused.At);
    const said = shownOrWithheld(Refused.Message);
    return at === undefined ? said : `${said} (${at})`;
}

function Offer({ release }: { release: MachinesRelease }) {
    const { flow } = useUpdate();
    const hello = useHello();
    const to = release.Latest ?? '';
    const notes = Openable.ofRelease(release);
    return (
        <Panel
            action={
                notes === undefined ? undefined : (
                    <OpenButton
                        held={NO_NOTES}
                        label={RELEASE_WORDS.whatChanged}
                        openable={notes}
                        variant="link"
                    />
                )
            }
            note="station and spwn, by the public installer"
            title={RELEASE_WORDS.update}
        >
            <div className={styles.acts}>
                <Button
                    onClick={() => {
                        flow.update({ executable: hello?.executable, to });
                    }}
                    variant="primary"
                >
                    {RELEASE_WORDS.update}
                </Button>
                <span className={styles.grow} />
                <Fine>
                    <Mono>{INSTALL_LINE}</Mono>
                </Fine>
            </div>
        </Panel>
    );
}

function Latest({ release }: { release: MachinesRelease }) {
    const check = useGesture('engine.refresh-release', {
        input: {},
        rereads: ['engine.read-release'],
        subject: undefined,
    });
    const refused = check.state.kind === 'failed' ? undefined : refusedOf(release);
    return (
        <Panel title="Engine">
            <Row
                below={<GestureOutcome gesture={check} />}
                sentence={sentenceOf(release, refused)}
                title="Latest release"
                trail={
                    <GestureButton
                        gesture={check}
                        label={RELEASE_WORDS.checkNow}
                        title="Read the latest release from GitHub now"
                    />
                }
            />
        </Panel>
    );
}

function sentenceOf(release: MachinesRelease, refused: string | undefined): string | undefined {
    const checked = checkedOf(release);
    if (refused === undefined) {
        return checked;
    }
    return checked === undefined ? refused : `${checked}${SEPARATOR}${refused}`;
}

function Release() {
    const reading = useRead('engine.read-release', {});
    const { flow, state } = useUpdate();
    if (reading.kind === 'reading') {
        return <Reading size="page" what="the latest release" />;
    }
    if (reading.kind === 'failed') {
        return <Unanswered error={reading.error} size="page" />;
    }
    const release = reading.out;
    return (
        <>
            <VerdictHead size="page" verdict={release.Head} />
            {state.kind === 'idle' ? null : <UpdatePanel flow={flow} state={state} wide />}
            {state.kind === 'idle' && release.Behind ? <Offer release={release} /> : null}
            <Latest release={release} />
        </>
    );
}

export function EnginePage() {
    const offered = useOffers('engine.read-release');
    return (
        <div className={page.page}>
            {offered ? (
                <Release />
            ) : (
                <NoVerdict
                    line="This engine reads no release: it is older than the call"
                    size="page"
                />
            )}
        </div>
    );
}
