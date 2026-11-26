// value: what the window never draws as a sentence: the old name, a path into its config folder, a command line.
const COMMAND_SHAPES: readonly RegExp[] = [
    /(?<![\w/-])station(?![\w/-])/iu,
    /~\/\.config/u,
    /(?:^|[\s`'"(:])(?:sudo|curl|launchctl|spwn|rm) [-\w<~/.$"]/u,
    /\bssh [\w.-]+@/u,
];

export function carriesCommand(text: string): boolean {
    return COMMAND_SHAPES.some((shape) => shape.test(text));
}

export function unlessCommand(text: string): string {
    return carriesCommand(text) ? '' : text;
}
