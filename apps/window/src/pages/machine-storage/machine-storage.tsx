import type { DiskReading, Target } from '@jterrazz/station-api';

import { Button, Screen } from '../../design/index';
import { useWatch } from '../../engine/react';
import type { Route } from '../../shell/route';
import { Freed, useFreeAll } from '../disk/disk-gestures';
import { ByApp, DiskPanel } from '../disk/disk-view';
import { HeldOutcomes } from '../kit/gestures';
import { Reading, Unanswered } from '../kit/readings';
import { Page } from '../page';

const FOOT = "Data is never freed here. Erasing an app's data is asked for by its name.";

type Go = (to: Route) => void;

function Inside({ go }: { go: Go | undefined }) {
    return (
        <>
            {'Inside this machine · on its Mac the whole disk is one file, read in '}
            {go === undefined ? (
                'Storage'
            ) : (
                <Button
                    onClick={() => {
                        go({ page: 'storage' });
                    }}
                    variant="link"
                >
                    Storage
                </Button>
            )}
        </>
    );
}

type ReadProps = {
    reading: DiskReading;
    name: string;
    target: Target | undefined;
    vm: boolean;
    go: Go | undefined;
};

function Read({ reading, name, target, vm, go }: ReadProps) {
    const freeAll = useFreeAll(target);
    return (
        <Screen>
            <Page>
                <DiskPanel
                    freeAll={freeAll}
                    name={name}
                    reading={reading}
                    sub={vm ? <Inside go={go} /> : undefined}
                    title="Storage"
                />
                <HeldOutcomes>
                    <Freed gesture={freeAll} />
                    <ByApp foot={FOOT} name={name} reading={reading} target={target} />
                </HeldOutcomes>
            </Page>
        </Screen>
    );
}

type StorageTabProps = {
    name: string;
    target: Target | undefined;
    vm?: boolean;
    go?: Go | undefined;
};

export function StorageTab({ name, target, vm = false, go }: StorageTabProps) {
    const disk = useWatch('disk.read', {}, target);
    if (disk.kind === 'failed') {
        return (
            <Page>
                <Unanswered error={disk.error} size="tab" />
            </Page>
        );
    }
    if (disk.kind === 'reading') {
        return (
            <Page>
                <Reading size="tab" what={`the disk of ${name}`} />
            </Page>
        );
    }
    return <Read go={go} name={name} reading={disk.out} target={target} vm={vm} />;
}
