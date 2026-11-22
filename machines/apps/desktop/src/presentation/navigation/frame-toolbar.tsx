import type { MachinesMacView } from '@jterrazz/station-api';
import type { ComponentType } from 'react';

import type { Found } from '../../domain/machines/machines';
import { crumbsOf, TITLES } from './crumbs';
import type { PlainPage } from './route';
import type { Router } from './router';
import { StaleMark } from './stale-mark';
import { Toolbar } from './toolbar';
import { actionOf } from './toolbar-action';

const NOWHERE = () => undefined;

export type FrameToolbarProps = {
    quiet: boolean;
    router: Router;
    found: Found | undefined;
    macs: readonly MachinesMacView[];
    ItemName: ComponentType<{ page: PlainPage; item: string }>;
    onToggleSidebar: () => void;
    onAddPassword: () => void;
    onNewMachine: () => void;
};

// value: the frame's toolbar: Home's crumb alone while the frame holds the no-engine panel, else the route's.
export function FrameToolbar(props: FrameToolbarProps) {
    const { quiet, router, found, macs, ItemName, onToggleSidebar } = props;
    const { route, go, back, forward, canGoBack, canGoForward } = router;
    if (quiet) {
        return (
            <Toolbar
                canGoBack={false}
                canGoForward={false}
                crumbs={[{ label: TITLES.home }]}
                onBack={NOWHERE}
                onForward={NOWHERE}
                onToggleSidebar={onToggleSidebar}
            />
        );
    }
    return (
        <Toolbar
            action={actionOf({
                addPassword: props.onAddPassword,
                macs,
                newMachine: props.onNewMachine,
                route,
            })}
            canGoBack={canGoBack}
            canGoForward={canGoForward}
            crumbs={crumbsOf({ found, go, ItemName, route })}
            mark={<StaleMark />}
            onBack={back}
            onForward={forward}
            onToggleSidebar={onToggleSidebar}
        />
    );
}
