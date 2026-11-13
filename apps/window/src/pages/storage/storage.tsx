import type { DiskReading, ImagesReport, MachinesMacView, Target } from '@jterrazz/station-api';
import { useState } from 'react';

import { Button, MachineIcon, Panel, Row, Switch } from '../../design/index';
import { diskGestureOf, machineKindOf } from '../../engine/provisional';
import { useReach, useRead, useWatch } from '../../engine/react';
import type { Reading } from '../../engine/react';
import { sentence } from '../../model/format';
import { targetOf } from '../../model/machines';
import { machineRoute } from '../../shell/route';
import type { Route } from '../../shell/route';
import { shortDay } from '../activity/when';
import { Freed, useFreeAll } from '../disk/disk-gestures';
import { DiskPanel, groupOf, GroupPanel } from '../disk/disk-view';
import { HeldOutcomes } from '../kit/gestures';
import { useIdentityOf } from '../kit/identity';
import { Absent, NoVerdict, Reading as ReadingVerdict, Unanswered } from '../kit/readings';
import { Page } from '../page';
import { chosenMac, imagesOn } from './chosen';
import { ImageLine } from './image-gesture';

import styles from '../page.module.css';

const BUILD_NEEDS = 'Building an image is not offered in the window yet';

type Go = (to: Route) => void;

type DiskOfProps = { reading: DiskReading; mac: string; target: Target | undefined };

function DiskRead({ reading, mac, target }: DiskOfProps) {
    const freeAll = useFreeAll(target);
    const erasable = groupOf(reading, (row) => diskGestureOf(row.Group)?.call === 'disk.erase');
    return (
        <>
            <DiskPanel freeAll={freeAll} name={mac} reading={reading} title="Disk" />
            <HeldOutcomes>
                <Freed gesture={freeAll} />
                {erasable === undefined ? null : <GroupPanel group={erasable} target={target} />}
            </HeldOutcomes>
        </>
    );
}

type DiskOfMacProps = { disk: Reading<DiskReading>; mac: string; target: Target | undefined };

function DiskOf({ disk, mac, target }: DiskOfMacProps) {
    if (disk.kind === 'failed') {
        return <Unanswered error={disk.error} size="tab" />;
    }
    if (disk.kind === 'reading') {
        return <ReadingVerdict size="tab" what={`the disk of ${mac}`} />;
    }
    return <DiskRead mac={mac} reading={disk.out} target={target} />;
}

function imageSentence(image: ImagesReport): string {
    const made = (image.VMs ?? []).length;
    return sentence([
        `built ${shortDay(image.Record.built_at) ?? image.Record.built_at}`,
        `${image.Record.size_gb} GB`,
        made === 0
            ? 'no machine made from it'
            : `${made} ${made === 1 ? 'machine' : 'machines'} made from it`,
    ]);
}

type ImagesProps = { images: Reading<null | readonly ImagesReport[]>; mac: MachinesMacView };

function Images({ images, mac }: ImagesProps) {
    const build = (
        <Button disabled title={BUILD_NEEDS}>
            Build a new one
        </Button>
    );
    const note = 'a machine built in advance; a new machine starts as a copy of it';
    if (images.kind === 'failed') {
        return (
            <Panel action={build} note={note} title="Images">
                <Unanswered error={images.error} size="panel" />
            </Panel>
        );
    }
    if (images.kind === 'reading') {
        return <Panel note="reading" title="Images" />;
    }
    const mine = imagesOn(images.out ?? [], mac.Name);
    return (
        <Panel action={build} note={note} title="Images">
            {mine.length === 0 ? (
                <p className={styles.none}>No image on {mac.Name}.</p>
            ) : (
                mine.map((image) => (
                    <ImageLine
                        image={image}
                        key={`${image.Record.name}@${image.Record.version}`}
                        mac={mac}
                        sentence={imageSentence(image)}
                    />
                ))
            )}
        </Panel>
    );
}

function Machines({ mac, go }: { mac: MachinesMacView; go: Go }) {
    const identityOf = useIdentityOf();
    return (
        <Panel note="what is inside one is on its own Storage tab" title="Machines">
            <Row
                lead={<MachineIcon kind="mac" />}
                onOpen={() => {
                    go(machineRoute(mac.Name, 'Storage'));
                }}
                sentence="the Mac itself"
                title={mac.Name}
                value={null}
            />
            {(mac.VMs ?? []).map((vm) => (
                <Row
                    key={vm.Name}
                    lead={<MachineIcon identity={identityOf(vm.Name)} kind={machineKindOf(vm)} />}
                    onOpen={() => {
                        go(machineRoute(vm.Name, 'Storage'));
                    }}
                    sentence={vm.Purpose}
                    title={vm.Name}
                    value={vm.SizeGB > 0 ? `${vm.SizeGB} GB` : null}
                />
            ))}
        </Panel>
    );
}

function OneMac({ mac, go }: { mac: MachinesMacView; go: Go }) {
    const target = targetOf({ mac });
    const disk = useWatch('disk.read', {}, target);
    const images = useRead('images.list', {});
    return (
        <>
            <DiskOf disk={disk} mac={mac.Name} target={target} />
            <HeldOutcomes>
                <Images images={images} mac={mac} />
            </HeldOutcomes>
            <Machines go={go} mac={mac} />
        </>
    );
}

export function StoragePage({ macs, go }: { macs: readonly MachinesMacView[]; go: Go }) {
    const reach = useReach();
    const [chosen, setChosen] = useState<string | undefined>(undefined);
    if (reach.kind === 'unanswered') {
        return (
            <Page>
                <Absent reach={reach} />
            </Page>
        );
    }
    const name = chosenMac(macs, chosen);
    const mac = macs.find((candidate) => candidate.Name === name);
    if (!mac) {
        return (
            <Page>
                <ReadingVerdict size="page" what="the Macs" />
            </Page>
        );
    }
    return (
        <Page>
            <NoVerdict size="page" />
            <Switch
                label="Which Mac"
                onChange={setChosen}
                options={macs.map((candidate) => ({ id: candidate.Name, label: candidate.Name }))}
                value={mac.Name}
            />
            <OneMac go={go} key={mac.Name} mac={mac} />
        </Page>
    );
}
