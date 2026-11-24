import type { DiskReading, Target } from '@jterrazz/station-api';

import { SEPARATOR } from '../../../../domain/separator';
import { useWatch } from '../../../dependencies/dependencies';
import { DiskCard } from '../../../features/disk/disk-card';
import { ByApp } from '../../../features/disk/disk-view';
import { Unanswered } from '../../../kit/readings';
import { Skeleton, SkeletonPanel } from '../../../kit/skeleton';
import type { Route } from '../../../navigation/route';
import { Screen } from '../../../ui/index';
import { Page } from '../../../ui/page';
import { DISK_TITLE } from '../../../words/words';

import styles from './machine-storage.module.css';

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
    go: Go | undefined;
};

function Read({ reading, name, target, go }: ReadProps) {
    return (
        <Screen>
            <Page>
                <DiskCard
                    leads
                    name={name}
                    reading={reading}
                    sub={<Inside go={go} />}
                    target={target}
                >
                    <ByApp name={name} reading={reading} target={target} />
                </DiskCard>
            </Page>
        </Screen>
    );
}

type StorageTabProps = {
    name: string;
    target: Target | undefined;
    go?: Go | undefined;
};

// value: a VM's Storage tab: what is inside its disk; a Mac's own disk is its section of the Storage page.
export function StorageTab({ name, target, go }: StorageTabProps) {
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
                    <SkeletonPanel title={DISK_TITLE} trail="fit" />
                </Skeleton>
            </Page>
        );
    }
    return <Read go={go} name={name} reading={disk.out} target={target} />;
}
