import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';

export const PRESENTATION = join(import.meta.dirname, '..', '..');

type Sheet = { path: string; source: string };

function sheetsUnder(dir: string): Sheet[] {
    return readdirSync(dir, { withFileTypes: true }).flatMap((entry): Sheet[] => {
        const path = join(dir, entry.name);
        if (entry.isDirectory()) {
            return sheetsUnder(path);
        }
        return path.endsWith('.css') ? [{ path, source: readFileSync(path, 'utf8') }] : [];
    });
}

// value: every style sheet of the window's presentation, read once with its path.
export const SHEETS: readonly Sheet[] = sheetsUnder(PRESENTATION);

// value: the closed list of the style sheets that draw the owner's orange, --you: a focus, a need, a warning or the
// value: disk running out. A new sheet that draws it is a deliberate edit of this list.
export const ORANGE_SHEETS: readonly string[] = [
    'features/disk/disk.module.css',
    'kit/login-sheet.module.css',
    'navigation/column.module.css',
    'navigation/path-bar.module.css',
    'navigation/space-switch.module.css',
    'screens/catalog/item.module.css',
    'screens/machine-files/machine-files.module.css',
    'screens/machine/overview/overview.module.css',
    'screens/machine/packages/packages.module.css',
    'screens/machine/storage/machine-storage.module.css',
    'screens/passwords/passwords.module.css',
    'ui/bar.module.css',
    'ui/canvas/canvas.module.css',
    'ui/chart.module.css',
    'ui/crumb-dots.module.css',
    'ui/document.css',
    'ui/dot-strip.module.css',
    'ui/fleet-board.module.css',
    'ui/form-field.module.css',
    'ui/guide.module.css',
    'ui/illustrations/illustration.module.css',
    'ui/nav-item.module.css',
    'ui/search-field.module.css',
    'ui/sheet-rise/strip.module.css',
    'ui/state-mark.module.css',
    'ui/switch.module.css',
    'ui/tile.module.css',
    'ui/value.module.css',
];

// value: the closed list of the style sheets that draw blue, --busy: an act in progress, a busy mark, a progress
// value: line or a running step. No chart, gauge, focus, link, selection or identity is blue.
export const BLUE_SHEETS: readonly string[] = [
    'screens/core/core.module.css',
    'ui/bar.module.css',
    'ui/dot-strip.module.css',
    'ui/progress-line.module.css',
    'ui/state-mark.module.css',
    'ui/tile.module.css',
];
