import type { MachinefilePlan } from '@jterrazz/station-api';
import { useState } from 'react';

import { useWatch } from '../../dependencies/dependencies';
import { PlanFileList } from '../../features/machinefile/files';
import { countOn, kindsOf, onShelf, planFilesOf } from '../../features/machinefile/model';
import type { Route } from '../../navigation/route';
import { Missing, Panel, Switch } from '../../ui/index';
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
    const planFiles = planFilesOf(plan);
    return (
        <Panel
            action={
                <Switch
                    label="Chapter"
                    onChange={setChapter}
                    options={CHAPTERS.map((id) => ({
                        count: id === 'Access' ? undefined : countOn(planFiles, id, kinds),
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
                <PlanFileList
                    look={{
                        descriptions: descriptionsOf(reports),
                        file,
                        kind: 'file',
                        open: (name) => {
                            go(catalogRouteOf(chapter, name));
                        },
                    }}
                    planFiles={onShelf(planFiles, chapter, kinds)}
                    shelf={chapter}
                />
            )}
        </Panel>
    );
}
