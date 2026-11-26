import type { CoreGuideGuide } from '@jterrazz/station-api';

import { machineRoute } from '../../navigation/route';
import type { Route } from '../../navigation/route';
import type { GuideAction, GuideProps } from '../../ui/guide';
import { wordOf } from '../../words/engine-words';

const MACHINE_PAGE = 'machine:';

// outside: a Guide's gesture names a page as the menu bar's pages are named, `machine:<name>` (apps/engine/machines/guide.go).
export function routeOfNav(nav: string): Route | undefined {
    const name = nav.startsWith(MACHINE_PAGE) ? nav.slice(MACHINE_PAGE.length) : '';
    return name === '' ? undefined : machineRoute(name);
}

export type Pressed = {
    action: GuideAction | undefined;
    quiet: GuideAction | undefined;
    onHide: () => void;
};

// invariant: the Guide's props are the engine's Guide field for field: the steps, the view, how it ended in its own
// invariant: word; only the gestures are the window's, each already a press.
export function guidePropsOf(
    guide: CoreGuideGuide,
    { action, quiet, onHide }: Pressed,
): GuideProps {
    const { view } = guide;
    return {
        act: guide.act,
        ended: guide.ended === undefined ? undefined : wordOf('CoreGuideEnded', guide.ended).Word,
        onHide,
        steps: guide.steps ?? [],
        subject: guide.subject,
        view: {
            action,
            headline: view.headline,
            illustration: view.illustration,
            kept: view.kept,
            move: view.move,
            progress: view.progress,
            prompt: view.prompt,
            quiet,
            sentence: view.sentence,
        },
    };
}
