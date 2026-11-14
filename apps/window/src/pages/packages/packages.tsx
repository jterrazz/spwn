import type {
    CallName,
    MachinefileChapterDrift,
    MachinefileDrift,
    MachinefileDriftInput,
    Target,
} from '@jterrazz/station-api';
import { useState } from 'react';

import { Screen, Verdict } from '../../design/index';
import type { Reading as Read } from '../../engine/react';
import { useRead } from '../../engine/react';
import { chapterNoun, fileJoinOf, verdictShown } from '../../engine/shown';
import { useGesture } from '../../engine/use-gesture';
import { SEPARATOR } from '../../separator';
import { useRouter } from '../../shell/router';
import { GestureButton, GestureOutcome } from '../kit/gestures';
import { NotGiven, NoVerdict, Reading } from '../kit/readings';
import { Page } from '../page';
import { Differs } from './differs';
import { Inventory } from './inventory';
import { headChapter, packageChapters } from './model';
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

function Upgrades({ target, name }: Aimed) {
    const upgrade = useGesture('machinefile.upgrade', {
        input: {},
        rereads: UPGRADE_REREADS,
        subject: name,
        target,
    });
    return (
        <>
            <NotGiven
                action={
                    <GestureButton
                        gesture={upgrade}
                        label="Upgrade now"
                        title={`Upgrade what the file of ${name} lists?`}
                        verb="Upgrade"
                    />
                }
                title="Upgrades"
                what="When it upgrades by itself, and its last run"
            />
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
            <Upgrades name={name} target={target} />
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
