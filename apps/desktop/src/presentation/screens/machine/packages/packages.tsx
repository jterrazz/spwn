import type {
    CallName,
    MachinefileChapterDrift,
    MachinefileDrift,
    MachinefileDriftInput,
    MachinefileLastRun,
    MachinefileUpgrades,
    Target,
} from '@jterrazz/station-api';
import { useState } from 'react';

import { clock } from '../../../../domain/format';
import { SEPARATOR } from '../../../../domain/separator';
import { shortDay } from '../../../../domain/when';
import type { Reading as Read } from '../../../dependencies/dependencies';
import { useRead } from '../../../dependencies/dependencies';
import { useGesture } from '../../../dependencies/use-gesture';
import { headChapter, packageChapters } from '../../../features/machinefile/model';
import { GestureButton, GestureOutcome } from '../../../kit/gestures';
import { NoVerdict, Reading } from '../../../kit/readings';
import { useRouter } from '../../../navigation/router';
import { Panel, Row, Screen, Verdict } from '../../../ui/index';
import { Page } from '../../../ui/page';
import { chapterNoun, fileJoinOf, verdictShown } from '../../../words/shown';
import { Differs } from './differs';
import { Inventory } from './inventory';
import { PlanView } from './plan';

import styles from './packages.module.css';

function FileName({ file }: { file: string }) {
    const { go } = useRouter();
    return (
        <button
            className={styles.fileLink}
            onClick={() => {
                go({ file, page: 'machine-files' });
            }}
            type="button"
        >
            {file}
        </button>
    );
}

function FileLink({ file }: { file: string }) {
    return (
        <>
            Its machine file is <FileName file={file} />
        </>
    );
}

function HeadTitle({ apps, file }: { apps: MachinefileChapterDrift; file: string }) {
    return file === '' ? (
        apps.Verdict.Sentence
    ) : (
        <>
            {apps.Verdict.Sentence}
            {fileJoinOf(apps.Verdict.Tone)}
            <FileName file={file} />.
        </>
    );
}

function chaptersMatch(drift: MachinefileDrift): string {
    return packageChapters(drift)
        .filter((chapter) => chapter.Listed > 0)
        .map(
            (chapter) =>
                `${chapter.Matching} of ${chapter.Listed} ${chapterNoun(chapter.Chapter, chapter.Listed)} match${SEPARATOR}`,
        )
        .join('');
}

function FileState({ drift, name }: { drift: Read<MachinefileDrift>; name: string }) {
    if (drift.kind === 'reading') {
        return <Reading size="tab" what={`the file ${name} follows`} />;
    }
    if (drift.kind === 'failed') {
        return <NoVerdict line="What the engine answered is below" size="tab" />;
    }
    const line = (
        <>
            {chaptersMatch(drift.out)}
            <FileLink file={drift.out.Plan.Name} />
        </>
    );
    const apps = headChapter(drift.out);
    if (apps === undefined) {
        return <NoVerdict line={line} size="tab" />;
    }
    return (
        <Verdict
            line={line}
            size="tab"
            title={<HeadTitle apps={apps} file={drift.out.Plan.Name} />}
            {...verdictShown(apps.Verdict)}
        />
    );
}

const UPGRADE_REREADS: readonly CallName[] = ['machinefile.drift', 'machinefile.plan'];

type Aimed = { target: Target | undefined; name: string };

type Asked = Aimed & { asked: MachinefileDriftInput };

function lastRunOf(last: MachinefileLastRun): string {
    const day = shortDay(last.At);
    return day === undefined ? '' : `${day}${SEPARATOR}${clock(last.At) ?? ''}`;
}

function UpgradeRows({ upgrades }: { upgrades: MachinefileUpgrades }) {
    const { Last: last } = upgrades;
    return (
        <>
            <Row size="slim" title={upgrades.Sentence} />
            {upgrades.File === '' ? null : <Row kind="File" size="slim" title={upgrades.File} />}
            {last === undefined ? null : (
                <Row
                    kind="Last run"
                    sentence={last.Verdict.Line}
                    size="slim"
                    {...(lastRunOf(last) === '' ? {} : { tag: lastRunOf(last) })}
                    title={last.Verdict.Sentence}
                />
            )}
        </>
    );
}

function UpgradeBody({ drift }: { drift: Read<MachinefileDrift> }) {
    if (drift.kind !== 'read') {
        return (
            <p className={styles.text}>
                {drift.kind === 'failed'
                    ? 'What the engine answered is below'
                    : 'Reading how it upgrades…'}
            </p>
        );
    }
    return <UpgradeRows upgrades={drift.out.Upgrades} />;
}

function Upgrades({ target, name, drift }: Aimed & { drift: Read<MachinefileDrift> }) {
    const upgrade = useGesture('machinefile.upgrade', {
        input: {},
        rereads: UPGRADE_REREADS,
        subject: name,
        target,
    });
    return (
        <>
            <Panel
                action={
                    <GestureButton
                        gesture={upgrade}
                        label="Upgrade now"
                        title={`Upgrade what the file of ${name} lists?`}
                        verb="Upgrade"
                    />
                }
                title="Upgrades"
            >
                <UpgradeBody drift={drift} />
            </Panel>
            <GestureOutcome gesture={upgrade} />
        </>
    );
}

function Setup({ target, name, asked, onPlan }: Asked & { onPlan: () => void }) {
    const drift = useRead('machinefile.drift', asked);
    return (
        <Page>
            <FileState drift={drift} name={name} />
            {drift.kind === 'read' ? <Differs drift={drift.out} onPlan={onPlan} /> : null}
            <Upgrades drift={drift} name={name} target={target} />
            <Inventory drift={drift} target={target} />
        </Page>
    );
}

export function PackagesTab({ name, target, asked }: Asked) {
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
                    asked={asked}
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
