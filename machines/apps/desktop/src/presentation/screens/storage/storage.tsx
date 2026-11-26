import type {
    DiskReading,
    DiskStationPart,
    ImagesReport,
    MachinesMacView,
    MachinesVM,
} from '@jterrazz/station-api';

import { bytes, sentence } from '../../../domain/format';
import { targetOf } from '../../../domain/machines/machines';
import type { Found } from '../../../domain/machines/machines';
import { shortDay } from '../../../domain/when';
import { useReach, useRead, useWatch } from '../../dependencies/dependencies';
import type { Reading } from '../../dependencies/dependencies';
import { DiskCard } from '../../features/disk/disk-card';
import { ByApp } from '../../features/disk/disk-view';
import { HeldOutcomes } from '../../kit/gestures';
import { useIdentityOf } from '../../kit/identity';
import { Absent, NoVerdict, Unanswered, VerdictHead } from '../../kit/readings';
import { SkeletonHead } from '../../kit/skeleton';
import { machineRoute } from '../../navigation/route';
import type { Route } from '../../navigation/route';
import { MachineIcon, MacSection, Panel, Row } from '../../ui/index';
import { Page } from '../../ui/page';
import { countOf } from '../../words/engine-words';
import { machineKindOf } from '../../words/provisional';
import { THE_MAC_ITSELF_WORD } from '../../words/words';
import { BuildingLine } from './building-line';
import { ImageLine } from './image-gesture';
import { imagesOn } from './images-on';

import styles from '../../ui/page.module.css';

type Go = (to: Route) => void;

type DiskOfProps = { reading: DiskReading; mac: MachinesMacView };

// value: a Mac's disk is its volume, what Machines holds beside it, then each app with its kinds and gestures.
function DiskRead({ reading, mac }: DiskOfProps) {
    const target = targetOf({ mac });
    return (
        <DiskCard leads={mac.ThisMac} name={mac.Name} reading={reading} target={target}>
            <ByApp name={mac.Name} reading={reading} target={target} />
        </DiskCard>
    );
}

type DiskOfMacProps = { disk: Reading<DiskReading>; mac: MachinesMacView };

function DiskOf({ disk, mac }: DiskOfMacProps) {
    if (disk.kind === 'failed') {
        return <Unanswered error={disk.error} size="tab" />;
    }
    if (disk.kind === 'reading') {
        return <SkeletonHead size="tab" what={`the disk of ${mac.Name}`} />;
    }
    return <DiskRead mac={mac} reading={disk.out} />;
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

function MacLine({ mac, station }: Omit<LineProps, 'go'>) {
    const caches = useCachesInside({ mac });
    return (
        <Row
            kind={caches.pill}
            lead={<MachineIcon kind="mac" />}
            sentence={sentence([THE_MAC_ITSELF_WORD, caches.said])}
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
        <Panel note="what is inside a VM is on its own Storage tab" title="Machines">
            <MacLine mac={mac} station={station} />
            {(mac.VMs ?? []).map((vm) => (
                <VmLine go={go} key={vm.Name} mac={mac} station={station} vm={vm} />
            ))}
        </Panel>
    );
}

function stationOf(disk: Reading<DiskReading>): DiskStationPart | undefined {
    return disk.kind === 'read' ? disk.out.Station : undefined;
}

type OneMacProps = { mac: MachinesMacView; go: Go };

function OneMac({ mac, go }: OneMacProps) {
    const disk = useWatch('disk.read', {}, targetOf({ mac }));
    const images = useRead('images.list', {});
    const station = stationOf(disk);
    return (
        <MacSection name={mac.Name}>
            {station === undefined ? (
                <NoVerdict size="tab" />
            ) : (
                <VerdictHead size="tab" verdict={station.Verdict} />
            )}
            <DiskOf disk={disk} mac={mac} />
            <HeldOutcomes>
                <Images images={images} mac={mac} />
            </HeldOutcomes>
            <Machines go={go} mac={mac} station={station} />
        </MacSection>
    );
}

// invariant: every Mac the engine lists has its section, in the listing's order, This Mac's first as listed.
export function StoragePage({ macs, go }: { macs: readonly MachinesMacView[]; go: Go }) {
    const reach = useReach();
    if (reach.kind === 'unanswered' && macs.length === 0) {
        return (
            <Page density="compact">
                <Absent reach={reach} />
            </Page>
        );
    }
    if (macs.length === 0) {
        return (
            <Page density="compact">
                <SkeletonHead size="page" what="the Macs" />
            </Page>
        );
    }
    return (
        <Page density="compact">
            {macs.map((mac) => (
                <OneMac go={go} key={mac.Name} mac={mac} />
            ))}
        </Page>
    );
}
