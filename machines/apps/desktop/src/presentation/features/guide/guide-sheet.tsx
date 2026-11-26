import type { CoreGuideGuide } from '@jterrazz/station-api';

import { macsOf } from '../../../domain/machines/machines';
import { useRead, useWatch } from '../../dependencies/dependencies';
import type { Route } from '../../navigation/route';
import { machineRoute } from '../../navigation/route';
import { Guide } from '../../ui/guide';
import { guidePropsOf } from './guide-of';
import { Pressing } from './press';

type Props = {
    act: string;
    go: (to: Route) => void;
    onClose: () => void;
    served: CoreGuideGuide | undefined;
};

// invariant: the sheet is the engine's Guide of the act the owner opened, read from the engine's one list and drawn
// invariant: as served; until the list holds the act it draws the Guide the engine served in the frame that named
// invariant: it, and it is read only while it is open.
export function GuideSheet({ act, go, onClose, served }: Props) {
    const guides = useRead('engine.guides', {});
    const machines = useWatch('machines.list', {});
    const listed =
        guides.kind === 'read'
            ? (guides.out.guides ?? []).find((one) => one.id === act)
            : undefined;
    const guide = listed ?? served;
    if (guide === undefined) {
        return null;
    }
    const macs = macsOf(machines.kind === 'read' ? machines.out : undefined);
    const onHide = () => {
        onClose();
        go(machineRoute(guide.subject));
    };
    const ground = { go, leave: onClose, macs };
    return (
        <Pressing {...ground} gesture={guide.view.action}>
            {(action) => (
                <Pressing {...ground} gesture={guide.view.quiet}>
                    {(quiet) => <Guide {...guidePropsOf(guide, { action, onHide, quiet })} />}
                </Pressing>
            )}
        </Pressing>
    );
}
