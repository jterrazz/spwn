import type {
    DiskReading,
    DiskStationPart,
    ImagesReport,
    MachinesMacView,
    MachinesVM,
    Target,
} from '@jterrazz/station-api';
import { useState } from 'react';

import { bytes, sentence } from '../../../domain/format';
import { targetOf } from '../../../domain/machines/machines';
import type { Found } from '../../../domain/machines/machines';
import { shortDay } from '../../../domain/when';
import { useReach, useRead, useWatch } from '../../dependencies/dependencies';
import type { Reading } from '../../dependencies/dependencies';
import { Freed, useFreeAll } from '../../features/disk/disk-gestures';
import { DiskPanel, groupOf, GroupPanel } from '../../features/disk/disk-view';
import { UnusedFoot, UnusedPanel } from '../../features/disk/unused';
import { HeldOutcomes } from '../../kit/gestures';
import { useIdentityOf } from '../../kit/identity';
import {
    Absent,
    NoVerdict,
    Reading as ReadingVerdict,
    Unanswered,
    VerdictHead,
} from '../../kit/readings';
import { machineRoute } from '../../navigation/route';
import type { Route } from '../../navigation/route';
import { MachineIcon, Panel, Row, Switch } from '../../ui/index';
import { Page } from '../../ui/page';
import { countOf } from '../../words/engine-words';
import { diskGestureOf, machineKindOf } from '../../words/provisional';
import { BuildingLine } from './building-line';
import { chosenMac, imagesOn } from './chosen';
import { ImageLine } from './image-gesture';

import styles from '../../ui/page.module.css';

type Go = (to: Route) => void;

type DiskOfProps = { reading: DiskReading; mac: string; target: Target | undefined };

function DiskRead({ reading, mac, target }: DiskOfProps) {
    const freeAll = useFreeAll(target);
    const station = reading.Station?.CanFree;
    const erasable = groupOf(reading, (row) => diskGestureOf(row.Group)?.call === 'disk.erase');
    return (
        <>
            <DiskPanel
                by="holds"
                freeAll={freeAll}
                name={mac}
                reading={reading}
                station={station === undefined ? null : <UnusedFoot canFree={station} />}
                sub={reading.Station?.Sentence}
                title="Disk"
            />
            <HeldOutcomes>
                <Freed gesture={freeAll} />
                {station === undefined ? null : <UnusedPanel canFree={station} />}
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
        made === 0 ? 'no machine made from it' : `${countOf('machine', made)} made from it`,
    ]);
}

type ImagesProps = { images: Reading<null | readonly ImagesReport[]>; mac: MachinesMacView };

function Images({ images, mac }: ImagesProps) {
    const note = 'a machine built in advance; a new machine starts as a copy of it';
    if (images.kind === 'failed') {
        return (
            <Panel note={note} title="Images">
                <Unanswered error={images.error} size="panel" />
            </Panel>
        );
    }
    if (images.kind === 'reading') {
        return <Panel note="reading" title="Images" />;
    }
    const mine = imagesOn(images.out ?? [], mac.Name);
    return (
        <Panel note={note} title="Images">
            {mine.length === 0 ? (
                <p className={styles.none}>No image on {mac.Name}.</p>
            ) : (
                mine.map((image) => {
                    const key = `${image.Record.name}@${image.Record.version}`;
                    return image.Building === null ? (
                        <ImageLine
                            image={image}
                            key={key}
                            mac={mac}
                            sentence={imageSentence(image)}
                        />
                    ) : (
                        <BuildingLine building={image.Building} image={image} key={key} />
                    );
                })
            )}
        </Panel>
    );
}

type Caches = { pill: null | string; said: string | undefined };

function useCachesInside(found: Found): Caches {
    const disk = useWatch('disk.read', {}, targetOf(found));
    const canFree = disk.kind === 'read' ? disk.out.CanFree : undefined;
    if (canFree === undefined) {
        return { pill: null, said: undefined };
    }
    return canFree.Call === undefined
        ? { pill: null, said: canFree.Sentence }
        : { pill: `${bytes(canFree.Bytes)} of caches inside`, said: undefined };
}

function sizeFrom(station: DiskStationPart | undefined, name: string): null | string {
    const held = station?.Machines?.find((machine) => machine.Name === name);
    return held === undefined ? null : bytes(held.Bytes);
}

type LineProps = { mac: MachinesMacView; station: DiskStationPart | undefined; go: Go };

function MacLine({ mac, station, go }: LineProps) {
    const caches = useCachesInside({ mac });
    return (
        <Row
            kind={caches.pill}
            lead={<MachineIcon kind="mac" />}
            onOpen={() => {
                go(machineRoute(mac.Name, 'Storage'));
            }}
            sentence={sentence(['the Mac itself', caches.said])}
            title={mac.Name}
            value={sizeFrom(station, mac.Name)}
        />
    );
}

function VmLine({ mac, vm, station, go }: LineProps & { vm: MachinesVM }) {
    const identityOf = useIdentityOf();
    const caches = useCachesInside({ mac, vm });
    return (
        <Row
            kind={caches.pill}
            lead={<MachineIcon identity={identityOf(vm.Name)} kind={machineKindOf(vm)} />}
            onOpen={() => {
                go(machineRoute(vm.Name, 'Storage'));
            }}
            sentence={sentence([vm.Purpose, caches.said])}
            title={vm.Name}
            value={sizeFrom(station, vm.Name)}
        />
    );
}

function Machines({ mac, station, go }: LineProps) {
    return (
        <Panel note="what is inside one is on its own Storage tab" title="Machines">
            <MacLine go={go} mac={mac} station={station} />
            {(mac.VMs ?? []).map((vm) => (
                <VmLine go={go} key={vm.Name} mac={mac} station={station} vm={vm} />
            ))}
        </Panel>
    );
}

type OneMacProps = { disk: Reading<DiskReading>; mac: MachinesMacView; go: Go };

function OneMac({ disk, mac, go }: OneMacProps) {
    const images = useRead('images.list', {});
    return (
        <>
            <DiskOf disk={disk} mac={mac.Name} target={targetOf({ mac })} />
            <HeldOutcomes>
                <Images images={images} mac={mac} />
            </HeldOutcomes>
            <Machines go={go} mac={mac} station={stationOf(disk)} />
        </>
    );
}

function stationOf(disk: Reading<DiskReading>): DiskStationPart | undefined {
    return disk.kind === 'read' ? disk.out.Station : undefined;
}

type ChosenProps = {
    mac: MachinesMacView;
    macs: readonly MachinesMacView[];
    choose: (name: string) => void;
    go: Go;
};

function ChosenMac({ mac, macs, choose, go }: ChosenProps) {
    const disk = useWatch('disk.read', {}, targetOf({ mac }));
    const station = stationOf(disk);
    return (
        <>
            {station === undefined ? (
                <NoVerdict size="page" />
            ) : (
                <VerdictHead size="page" verdict={station.Verdict} />
            )}
            <Switch
                label="Which Mac"
                onChange={choose}
                options={macs.map((candidate) => ({ id: candidate.Name, label: candidate.Name }))}
                value={mac.Name}
            />
            <OneMac disk={disk} go={go} mac={mac} />
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
            <ChosenMac choose={setChosen} go={go} key={mac.Name} mac={mac} macs={macs} />
        </Page>
    );
}
