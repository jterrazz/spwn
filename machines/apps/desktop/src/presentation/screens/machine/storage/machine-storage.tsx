import type { DiskReading, Target } from '@jterrazz/station-api';

import { SEPARATOR } from '../../../../domain/separator';
import { useWatch } from '../../../dependencies/dependencies';
import { Freed, useFreeAll } from '../../../features/disk/disk-gestures';
import { ByApp, DiskPanel } from '../../../features/disk/disk-view';
import { HeldOutcomes } from '../../../kit/gestures';
import { Unanswered } from '../../../kit/readings';
import { Skeleton, SkeletonPanel } from '../../../kit/skeleton';
import type { Route } from '../../../navigation/route';
import { Screen } from '../../../ui/index';
import { Page } from '../../../ui/page';

import styles from './machine-storage.module.css';

const TITLE = 'Storage';

const FOOT = "Data and settings are never freed here. Erasing an app's data is asked for by name.";

type Go = (to: Route) => void;

function Inside({ go }: { go: Go | undefined }) {
    return (
        <>
            {`Inside this machine${SEPARATOR}on its Mac the whole disk is one file, read in `}
            {go === undefined ? (
                'Storage'
            ) : (
                <button
                    className={styles.storageLink}
                    onClick={() => {
                        go({ page: 'storage' });
                    }}
                    type="button"
                >
                    Storage
                </button>
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
                    by="kinds"
                    freeAll={freeAll}
                    name={name}
                    reading={reading}
                    sub={vm ? <Inside go={go} /> : reading.Station?.Itself}
                    title={TITLE}
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
                <Skeleton what={`the disk of ${name}`}>
                    <SkeletonPanel count={3} title={TITLE} trail="fit" />
                </Skeleton>
            </Page>
        );
    }
    return <Read go={go} name={name} reading={disk.out} target={target} vm={vm} />;
}
