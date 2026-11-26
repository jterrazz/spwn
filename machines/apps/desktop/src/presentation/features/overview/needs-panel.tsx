import type { MachinesMacView, OverviewNeed } from '@jterrazz/station-api';

import { isNeedsYou } from '../../../domain/overview/upkeep';
import type { Route } from '../../navigation/route';
import { Panel } from '../../ui/index';
import { needState } from '../../words/shown';
import { NeedLine } from './need-gesture';
import { routeOfNeed } from './needs';

import pageStyles from '../../ui/page.module.css';

type Go = (to: Route) => void;

type NeedsProps = {
    needs: readonly OverviewNeed[];
    title: string;
    macs: readonly MachinesMacView[];
    go: Go;
    none?: string;
    on?: string;
    note?: string;
    standsBack?: boolean;
};

function keyOf(need: OverviewNeed): string {
    return `${need.Machine}/${need.Section}/${need.Subject}`;
}

// invariant: a need the engine no longer lists after its gesture takes its outcome to the page's HeldOutcomes,
// invariant: so every page that draws this panel holds it inside one.
// invariant: a page whose bar leads with its own button asks the panel to stand back, so no need draws a second
// invariant: primary on the screen.
export function Needs({ needs, title, macs, go, none, note, standsBack = false }: NeedsProps) {
    if (needs.length === 0 && none === undefined) {
        return null;
    }
    const bright = standsBack
        ? -1
        : needs.findIndex((need) => isNeedsYou(need) && needState(need).tone === 'bad');
    return (
        <Panel note={note ?? String(needs.length)} title={title}>
            {needs.length === 0 ? (
                <p className={pageStyles.missing}>{none}</p>
            ) : (
                needs.map((need, index) => (
                    <NeedLine
                        key={keyOf(need)}
                        macs={macs}
                        need={need}
                        onOpen={() => {
                            go(routeOfNeed(need));
                        }}
                        primary={index === bright}
                    />
                ))
            )}
        </Panel>
    );
}
