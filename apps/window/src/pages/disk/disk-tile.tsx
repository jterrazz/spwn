import type { Target } from '@jterrazz/station-api';

import { Bar, Tile } from '../../design/index';
import { useWatch } from '../../engine/react';
import { bytes, sentence } from '../../model/format';
import { NotGivenTile, split, Unknown } from '../activity/tiles';

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
    const { Free, Total } = disk.out;
    if (Total <= 0 || Free < 0) {
        return <NotGivenTile label={label} />;
    }
    const { value, unit } = split(bytes(shows === 'free' ? Free : Total - Free));
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
                    percent={((Total - Free) / Total) * 100}
                    width="fixed"
                />
            }
        />
    );
}
