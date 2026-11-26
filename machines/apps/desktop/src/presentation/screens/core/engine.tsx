import type { MachinesMacView, MachinesRelease } from '@jterrazz/station-api';

import { SEPARATOR } from '../../../domain/separator';
import { dayAndClock } from '../../../domain/when';
import { useHello, useOffers, useReach, useRead, useUpdate } from '../../dependencies/dependencies';
import { useGesture } from '../../dependencies/use-gesture';
import { GestureButton, GestureOutcome } from '../../kit/gestures';
import { Openable } from '../../kit/openable';
import { OpenButton } from '../../kit/opens';
import { Absent, NoVerdict, Unanswered, VerdictHead } from '../../kit/readings';
import { Skeleton, SkeletonHead, SkeletonPanel } from '../../kit/skeleton';
import { Button, MacSection, Panel, Row, Value, VerdictSkeleton } from '../../ui/index';
import { Page } from '../../ui/page';
import { shownOrWithheld } from '../../ui/secret';
import { engineShown, macState } from '../../words/shown';
import type { Shown } from '../../words/shown';
import { RELEASE_WORDS } from '../../words/words';
import { UpdatePanel } from './update';

import styles from './core.module.css';

const NO_NOTES = 'The engine names no release notes';

const ENGINE_TITLE = 'Engine';

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
            note="the public installer installs Spwn"
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
            </div>
        </Panel>
    );
}

function Latest({ release, mac }: { release: MachinesRelease; mac: MachinesMacView }) {
    const check = useGesture('engine.refresh-release', {
        input: {},
        rereads: ['engine.read-release'],
        subject: undefined,
    });
    const refused = check.state.kind === 'failed' ? undefined : refusedOf(release);
    return (
        <Panel title={ENGINE_TITLE}>
            <EngineRow mac={mac} />
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

function stateOf(mac: MachinesMacView): Shown {
    const { State } = mac.Engine;
    return State === undefined ? macState(mac) : engineShown('EngineReadState', State);
}

function EngineRow({ mac }: { mac: MachinesMacView }) {
    const { Version } = mac.Engine;
    return (
        <Row
            state={stateOf(mac)}
            title={ENGINE_TITLE}
            value={Version === '' ? null : <Value>{Version}</Value>}
        />
    );
}

function FarEngine({ mac }: { mac: MachinesMacView }) {
    return (
        <Panel title={ENGINE_TITLE}>
            <EngineRow mac={mac} />
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

function Release({ mac }: { mac: MachinesMacView }) {
    const reading = useRead('engine.read-release', {});
    const { flow, state } = useUpdate();
    if (reading.kind === 'reading') {
        return (
            <Skeleton what="the latest release">
                <VerdictSkeleton size="tab" />
                <SkeletonPanel title={ENGINE_TITLE} trail="fit" />
            </Skeleton>
        );
    }
    if (reading.kind === 'failed') {
        return (
            <>
                <Unanswered error={reading.error} size="tab" />
                <FarEngine mac={mac} />
            </>
        );
    }
    const release = reading.out;
    return (
        <>
            <VerdictHead size="tab" verdict={release.Head} />
            {state.kind === 'idle' ? null : <UpdatePanel flow={flow} state={state} wide />}
            {state.kind === 'idle' && release.Behind ? <Offer release={release} /> : null}
            <Latest mac={mac} release={release} />
        </>
    );
}

// invariant: the release is This Mac's engine's to read and to update; another Mac's section says its engine's
// invariant: state and version as the listing gives them.
function ThisMac({ mac }: { mac: MachinesMacView }) {
    const offered = useOffers('engine.read-release');
    return offered ? (
        <Release mac={mac} />
    ) : (
        <>
            <NoVerdict line="This engine reads no release: it is older than the call" size="tab" />
            <FarEngine mac={mac} />
        </>
    );
}

export function EnginePage({ macs }: { macs: readonly MachinesMacView[] }) {
    const reach = useReach();
    if (reach.kind === 'unanswered' && macs.length === 0) {
        return (
            <Page>
                <Absent reach={reach} />
            </Page>
        );
    }
    if (macs.length === 0) {
        return (
            <Page>
                <SkeletonHead size="page" what="the Macs" />
            </Page>
        );
    }
    return (
        <Page>
            {macs.map((mac) => (
                <MacSection key={mac.Name} name={mac.Name}>
                    {mac.ThisMac ? <ThisMac mac={mac} /> : <FarEngine mac={mac} />}
                </MacSection>
            ))}
        </Page>
    );
}
