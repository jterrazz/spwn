import type { ActivityReport, MachinesMacView } from '@jterrazz/station-api';

import { Bar, Tile } from '../../design/index';
import { useWatch } from '../../engine/react';
import { AWAKE_ON_TONE } from '../../engine/tones';
import { AWAKE_WORDS, POWER_SOURCE_WORDS } from '../../engine/words';
import { bytes, clock, duration, sentence } from '../../model/format';
import { useActivity } from '../activity/reading';
import { CpuTile, NotGivenTile, split, Unknown, unknownOf } from '../activity/tiles';
import { Tiles } from '../page';

const NS_PER_MS = 1_000_000;

const LABELS = {
    awake: 'Awake, this Mac',
    cpu: 'CPU, this Mac',
    disk: 'Disk free, this Mac',
    memory: 'Memory free',
} as const;

function DiskFreeTile({ report }: { report: ActivityReport }) {
    const { disk_free: free, disk_total: total } = report.last.sample;
    if (free === undefined || total === undefined || total <= 0) {
        return <NotGivenTile label={LABELS.disk} />;
    }
    const { value, unit } = split(bytes(free));
    return (
        <Tile
            foot={`of ${bytes(total)}`}
            label={LABELS.disk}
            unit={unit}
            value={value}
            viz={
                <Bar
                    label={`${LABELS.disk}: used of its total`}
                    percent={((total - free) / total) * 100}
                    width="fixed"
                />
            }
        />
    );
}

function leftOf(until: string, now: string): number | undefined {
    const left = Date.parse(until) - Date.parse(now);
    return Number.isNaN(left) || left <= 0 ? undefined : left * NS_PER_MS;
}

function AwakeTile() {
    const awake = useWatch('awake.read', {});
    if (awake.kind !== 'read') {
        return (
            <Unknown
                label={LABELS.awake}
                state={awake.kind === 'failed' ? 'no-answer' : 'reading'}
            />
        );
    }
    const { NeverSleeps, Session, Now, Reading } = awake.out;
    const source = POWER_SOURCE_WORDS[Reading.Source] ?? Reading.Source;
    const left = Session === null ? undefined : leftOf(Session.until, Now);
    if (NeverSleeps || left === undefined) {
        return (
            <Tile
                foot={sentence([NeverSleeps ? AWAKE_WORDS.neverSleeps : undefined, source])}
                label={LABELS.awake}
                value={NeverSleeps ? AWAKE_WORDS.on : AWAKE_WORDS.off}
            />
        );
    }
    const { value, unit } = split(duration(left));
    return (
        <Tile
            aside={{ text: AWAKE_WORDS.on, tone: AWAKE_ON_TONE }}
            foot={sentence([`until ${clock(Session?.until) ?? ''}`, source])}
            label={LABELS.awake}
            unit={`${unit} left`}
            value={value}
        />
    );
}

export function HomeTiles({ mac }: { mac: MachinesMacView }) {
    const { readers, shown } = useActivity({ mac }, '1h');
    const unknown = unknownOf(shown);
    const report = shown?.kind === 'read' ? shown.out : undefined;
    return (
        <>
            {readers}
            <Tiles>
                <NotGivenTile label={LABELS.memory} />
                {unknown !== undefined || report === undefined ? (
                    <Unknown label={LABELS.cpu} state={unknown ?? 'reading'} />
                ) : (
                    <CpuTile label={LABELS.cpu} name={mac.Name} report={report} />
                )}
                <AwakeTile />
                {unknown !== undefined || report === undefined ? (
                    <Unknown label={LABELS.disk} state={unknown ?? 'reading'} />
                ) : (
                    <DiskFreeTile report={report} />
                )}
            </Tiles>
        </>
    );
}
