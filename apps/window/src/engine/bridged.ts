import type { CallName } from '@jterrazz/station-api';

// invariant: the calls the window sends, and the only ones its bridge forwards: `src-tauri/build.rs` reads this
// invariant: list, one quoted name a line, and the bridge refuses any other. `ReadName` and `WatchName` admit no
// invariant: other name, so a call sent off the list fails the compile, and `bridged.test.ts` fails a name no
// invariant: source sends.
export const BRIDGED = [
    'accounts.list',
    'accounts.share',
    'accounts.take-back',
    'activity.read',
    'apps.list',
    'apps.update',
    'awake.read',
    'background.remove',
    'disk.erase',
    'disk.free',
    'disk.move',
    'disk.read',
    'engine.events',
    'engine.read',
    'engine.start',
    'engine.watch',
    'images.delete',
    'images.list',
    'machinefile.apply',
    'machinefile.drift',
    'machinefile.explain',
    'machinefile.plan',
    'machinefile.upgrade',
    'machines.create-server',
    'machines.create-workspace',
    'machines.delete',
    'machines.list',
    'machines.log',
    'machines.restart',
    'machines.ssh',
    'machines.start',
    'machines.stop',
    'overview.read',
    'ports.declare',
    'ports.list',
    'ports.share',
    'ports.stop-sharing',
    'settings.list',
    'settings.matrix',
] as const satisfies readonly CallName[];

export type Bridged = (typeof BRIDGED)[number];
