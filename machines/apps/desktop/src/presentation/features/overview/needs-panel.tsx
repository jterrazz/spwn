import type { MachinesMacView, OverviewNeed, OverviewUpkeep } from '@jterrazz/station-api';

import type { Route } from '../../navigation/route';
import { Panel } from '../../ui/index';
import { leadingNeed } from '../../words/shown';
import { NeedLine } from './need-gesture';
import { routeOfNeed } from './needs';

import pageStyles from '../../ui/page.module.css';

type Go = (to: Route) => void;

// value: `names` is the state the title says, so a row in that state draws its mark without the word.
type NeedsProps = {
    needs: readonly OverviewNeed[];
    title: string;
    names: OverviewUpkeep;
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
export function Needs({
    needs,
    title,
    names,
    macs,
    go,
    none,
    note,
    standsBack = false,
}: NeedsProps) {
    if (needs.length === 0 && none === undefined) {
        return null;
    }
    const bright = standsBack ? -1 : leadingNeed(needs);
    return (
        <Panel note={note ?? String(needs.length)} title={title}>
            {needs.length === 0 ? (
                <p className={pageStyles.missing}>{none}</p>
            ) : (
                needs.map((need, index) => (
                    <NeedLine
                        key={keyOf(need)}
                        macs={macs}
                        named={need.Mark === names}
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
