export interface LogLine {
    timestamp: string;
    source: string;
    message: string;
}

/** The last load of a world's log that settled, and whether it settled during the panel's current opening. */
export interface SettledLog {
    worldId: string;
    lines: LogLine[];
    thisOpening: boolean;
}

/** What the logs panel shows: the lines last read for this world, and loading while no load of it has settled since the panel opened. */
export function logsPanelOf(
    settled: null | SettledLog,
    worldId: string,
    open: boolean,
): { lines: LogLine[]; loading: boolean } {
    const ofThisWorld = settled?.worldId === worldId;
    return {
        lines: ofThisWorld ? settled.lines : [],
        loading: open && !(ofThisWorld && settled.thisOpening),
    };
}

/** The settled load once the panel closes: its lines kept, but no longer read during an opening. */
export function closedLog(settled: null | SettledLog): null | SettledLog {
    return settled === null ? null : { ...settled, thisOpening: false };
}
