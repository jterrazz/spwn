import { Verdict } from '../../design/index';
import { Page } from '../page';
import { ChoiceCards, DialogSample, Fields } from './forms';
import { Confirms, GestureStates } from './gestures';
import { Buttons, IconButtons, StatePills, Switches, Tags } from './marks';
import { Measures } from './measures';
import { AppIcons, Identities, NavItems } from './navigation';
import { CodeBlocks, FactsSample, PanelsAndRows } from './panels';

export default function Gallery() {
    return (
        <Page>
            <Verdict
                line="The proof surface of the design system"
                size="page"
                title="Every component, every variant."
                tone="ok"
                word="Up"
            />
            <Verdict
                line="Its line under it, with a dot that carries its word"
                size="tab"
                title="A tab opens on this verdict."
                tone="warn"
                word="Needs you"
            />
            <StatePills />
            <Tags />
            <Buttons />
            <IconButtons />
            <Switches />
            <NavItems />
            <Identities />
            <AppIcons />
            <Measures />
            <PanelsAndRows />
            <FactsSample />
            <CodeBlocks />
            <Fields />
            <ChoiceCards />
            <DialogSample />
            <Confirms />
            <GestureStates />
        </Page>
    );
}
