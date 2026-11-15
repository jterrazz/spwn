import { carriedApps, carriedSettings } from '../domain/catalog/carried';
import { useRead, useWatch } from '../engine/react';

// value: how many packages of each shelf some machine carries, counted from the wire's own matrix rows; the reads
// value: are the ones the catalog pages make, shared by the engine's store.
export type Carried = {
    apps: number | undefined;
    skills: number | undefined;
    settings: number | undefined;
};

export function useCarried(): Carried {
    const apps = useWatch('apps.list', {});
    const settings = useWatch('settings.list', {});
    const appsHeld = useRead('apps.matrix', {});
    const settingsHeld = useRead('settings.matrix', {});
    const appsRead =
        apps.kind === 'read' && appsHeld.kind === 'read'
            ? { reports: apps.out ?? [], rows: appsHeld.out.Rows ?? [] }
            : undefined;
    const settingsRead =
        settings.kind === 'read' && settingsHeld.kind === 'read'
            ? { rows: settingsHeld.out.Rows ?? [], settings: settings.out ?? [] }
            : undefined;
    return {
        apps: appsRead && carriedApps('apps', appsRead.reports, appsRead.rows).length,
        settings: settingsRead && carriedSettings(settingsRead.settings, settingsRead.rows).length,
        skills: appsRead && carriedApps('skills', appsRead.reports, appsRead.rows).length,
    };
}
