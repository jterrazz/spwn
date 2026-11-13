import type { MachinesMacView, OverviewNeed } from '@jterrazz/station-api';

import { Panel } from '../../design/index';
import type { Route } from '../../shell/route';
import { NeedLine } from './need-gesture';
import { routeOfNeed } from './needs';

import pageStyles from '../page.module.css';

type Go = (to: Route) => void;

type NeedsProps = {
    needs: readonly OverviewNeed[];
    title: string;
    macs: readonly MachinesMacView[];
    go: Go;
    none?: string;
    on?: string;
    note?: string;
};

function keyOf(need: OverviewNeed): string {
    return `${need.Machine}/${need.Section}/${need.Subject}`;
}

// invariant: a need the engine no longer lists after its gesture takes its outcome to the page's HeldOutcomes,
// invariant: so every page that draws this panel holds it inside one.
export function Needs({ needs, title, macs, go, none, note }: NeedsProps) {
    if (needs.length === 0 && none === undefined) {
        return null;
    }
    return (
        <Panel note={note ?? String(needs.length)} title={title}>
            {needs.length === 0 ? (
                <p className={pageStyles.missing}>{none}</p>
            ) : (
                needs.map((need) => (
                    <NeedLine
                        key={keyOf(need)}
                        macs={macs}
                        need={need}
                        onOpen={() => {
                            go(routeOfNeed(need));
                        }}
                    />
                ))
            )}
        </Panel>
    );
}
