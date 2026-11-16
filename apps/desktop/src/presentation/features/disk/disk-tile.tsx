import type { Target } from '@jterrazz/station-api';

import { bytes, sentence } from '../../../domain/format';
import { useWatch } from '../../dependencies/dependencies';
import { NotGivenTile, split, Unknown } from '../../kit/tile-states';
import { Bar, Tile } from '../../ui/index';

type Shows = 'free' | 'used';

export type DiskTileProps = {
    label: string;
    shows: Shows;
    target: Target | undefined;
    held?: number | undefined;
};

// value: a disk as disk.read measures it, its free or its used space against its total. `held` is the sum of
// value: the disk sizes of the machines a Mac holds, drawn as they are listed.
export function DiskTile({ label, shows, target, held }: DiskTileProps) {
    const disk = useWatch('disk.read', {}, target);
    if (disk.kind !== 'read') {
        return <Unknown label={label} state={disk.kind === 'failed' ? 'no-answer' : 'reading'} />;
    }
    const { Free, Total, Used } = disk.out;
    if (Total <= 0 || Free < 0) {
        return <NotGivenTile label={label} />;
    }
    const { value, unit } = split(bytes(shows === 'free' ? Free : Used));
    return (
        <Tile
            foot={sentence([
                `of ${bytes(Total)}`,
                held === undefined || held <= 0 ? undefined : `machines hold ${bytes(held)}`,
            ])}
            label={label}
            unit={unit}
            value={value}
            viz={
                <Bar
                    label={`${label}: used of its total`}
                    percent={(Used / Total) * 100}
                    width="fixed"
                />
            }
        />
    );
}
