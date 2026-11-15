import type { MachinefilePlan } from '@jterrazz/station-api';
import { useState } from 'react';

import { Missing, Panel, Switch } from '../../design/index';
import { useWatch } from '../../engine/react';
import type { Route } from '../../shell/route';
import { LayerList } from '../packages/layers';
import { countOn, kindsOf, layersOf, onShelf } from '../packages/model';
import { catalogRouteOf, CHAPTERS, descriptionsOf } from './model';
import type { Chapter } from './model';

export function Chapters({
    plan,
    file,
    go,
}: {
    plan: MachinefilePlan;
    file: string;
    go: (to: Route) => void;
}) {
    const [chapter, setChapter] = useState<Chapter>('Apps');
    const apps = useWatch('apps.list', {});
    const reports = apps.kind === 'read' ? (apps.out ?? []) : [];
    const kinds = kindsOf(reports);
    const layers = layersOf(plan);
    return (
        <Panel
            action={
                <Switch
                    label="Chapter"
                    onChange={setChapter}
                    options={CHAPTERS.map((id) => ({
                        count: id === 'Access' ? undefined : countOn(layers, id, kinds),
                        id,
                        label: id,
                    }))}
                    surface="panel"
                    value={chapter}
                />
            }
            note="the boxed values are the ones you set"
            title="What it puts on a machine"
        >
            {chapter === 'Access' ? (
                <Missing>The accounts it shares and the ports it opens</Missing>
            ) : (
                <LayerList
                    layers={onShelf(layers, chapter, kinds)}
                    look={{
                        descriptions: descriptionsOf(reports),
                        file,
                        kind: 'file',
                        open: (name) => {
                            go(catalogRouteOf(chapter, name));
                        },
                    }}
                    shelf={chapter}
                />
            )}
        </Panel>
    );
}
