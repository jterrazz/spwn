import { TOOLS } from './tools-catalog.data';

export type ToolStatus = 'available' | 'planned';

export interface SkillFile {
    name: string;
    content: string;
}

export interface ToolDef {
    name: string;
    description: string;
    provides: string;
    useWhen: string;
    dependencies: string[];
    verify: string[];
    status: ToolStatus;
    skills: SkillFile[];
}

export function getToolByName(name: string): ToolDef | undefined {
    return TOOLS.find((t) => t.name === name);
}

/** Get the slug from a tool name (e.g. "spwn:qmd" gives "qmd"). */
export function toolSlug(name: string): string {
    return name.replace('spwn:', '');
}
