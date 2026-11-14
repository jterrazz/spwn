import type {
    AppsReport,
    MachinefileCallsExplainOut,
    MachinefileOrigin,
} from '@jterrazz/station-api';

import { IDENTITIES } from '../../design/index';
import type { Identity } from '../../design/index';
import { SEPARATOR } from '../../design/separator';
import { wordOfSent } from '../../engine/engine-words';
import { SHELVES } from '../packages/model';

export const CHAPTERS = [...SHELVES, 'Access'] as const;

export type Chapter = (typeof CHAPTERS)[number];

export type Extended = { name: string; identity: Identity };

export function filesOf(explain: MachinefileCallsExplainOut): readonly MachinefileOrigin[] {
    const files = explain.Files ?? [];
    const seen = new Set<string>();
    return files.filter((file) => {
        if (file.Name === '' || seen.has(file.Name)) {
            return false;
        }
        seen.add(file.Name);
        return true;
    });
}

export function descriptionsOf(reports: readonly AppsReport[]): ReadonlyMap<string, string> {
    return new Map(reports.map((report) => [report.App.Name, report.App.Description]));
}

export function factsOf(
    explain: MachinefileCallsExplainOut,
    file: string,
): { makes: string; system: string } {
    const values = (explain.Values ?? []).filter(
        (value) => file === explain.Name || value.Origin.Name === file,
    );
    const text = new Map(values.map((value) => [value.Key, value.Text]));
    const os = text.get('machine.system.os') ?? '';
    const named = [os === '' ? '' : wordOfSent('CatalogOS', os), text.get('machine.system.version')]
        .filter((part) => part !== undefined && part !== '')
        .join(' ');
    const arch = wordOfSent('CatalogArch', text.get('machine.system.arch') ?? '');
    return {
        makes: text.get('machine.kind') ?? '',
        system: [named, arch].filter((part) => part !== '').join(SEPARATOR),
    };
}

export function extendedBy(explain: MachinefileCallsExplainOut, file: string): readonly Extended[] {
    if (file !== explain.Name) {
        return [];
    }
    const files = filesOf(explain);
    const at = files.findIndex((candidate) => candidate.Name === file);
    return (at === -1 ? files : files.slice(0, at)).map((candidate, index) => ({
        identity: IDENTITIES[index % IDENTITIES.length] ?? 'system',
        name: candidate.Name,
    }));
}
