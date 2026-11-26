import type { Recording } from '../../../../infrastructure/outbound/engine-replay/replay';

export const RESTART = {
    Args: { Name: 'infra' },
    Broken: true,
    Call: 'machines.restart',
    Command: '',
    Confirm: '',
    InTerminal: false,
    Machine: 'infra',
    Mark: 'down',
    Noun: 'Overview',
    Object: '',
    On: 'mac-mini',
    Pane: '',
    Path: '',
    Section: 'overview',
    Severity: 'broken',
    Subject: 'infra',
    Text: 'infra check fails',
    Upkeep: 'needs-you',
    Verb: 'restart',
};

export const TRY_AGAIN = {
    Args: { VM: 'api' },
    Broken: true,
    Call: 'machines.start',
    Command: 'station machines start api',
    Confirm: '',
    InTerminal: false,
    Machine: 'api',
    Mark: 'down',
    Noun: 'Overview',
    Object: '',
    On: 'macbook-pro',
    Pane: '',
    Path: '',
    Section: 'overview',
    Severity: 'broken',
    Subject: 'api',
    Text: 'api was not created: it failed at boot',
    Upkeep: 'needs-you',
    Verb: 'try-again',
};

export const DELETE = {
    ...TRY_AGAIN,
    Args: { VM: 'api' },
    Call: 'machines.delete',
    Command: 'station machines delete api',
    Confirm: 'api',
    Text: 'api was not created: it failed at SSH',
    Verb: 'delete',
};

export const UPGRADE = {
    ...TRY_AGAIN,
    Machine: 'macbook-pro',
    Subject: 'macbook-pro',
    Mark: 'needs-you',
    Args: { Machinefile: 'macbook-pro' },
    Call: 'machinefile.upgrade',
    Command: 'station machinefile update',
    Noun: 'Apps',
    Section: 'apps',
    Text: "the night's upgrade of macbook-pro failed: it stopped",
    Verb: 'upgrade',
};

export const PATH_ONLY = {
    Args: null,
    Broken: false,
    Call: '',
    Command: '',
    Confirm: '',
    InTerminal: false,
    Machine: 'macbook-pro',
    Mark: 'needs-you',
    Noun: 'Settings',
    Object: '',
    On: 'macbook-pro',
    Pane: '',
    Path: 'System Settings > Network > Firewall',
    Section: 'settings',
    Severity: 'needs-a-look',
    Subject: 'firewall',
    Text: 'Firewall: off',
    Upkeep: 'needs-you',
    Verb: 'fix',
};

export const PANE = {
    ...PATH_ONLY,
    Object: 'Firewall settings',
    Pane: 'x-apple.systempreferences:com.apple.Network-Settings.extension?Firewall',
    Verb: 'open-settings',
};

export const FIX_LINE = {
    ...PATH_ONLY,
    Command: 'sudo launchctl bootout system/com.example.vmnetd',
    InTerminal: true,
    Path: '',
    Subject: 'com.example.vmnetd',
    Text: 'com.example.vmnetd is left over',
    Verb: 'in-terminal',
};

function recordOf(value: unknown): Record<string, unknown> | undefined {
    return typeof value === 'object' && value !== null
        ? Object.fromEntries(Object.entries(value))
        : undefined;
}

function needed(answer: unknown, list: readonly object[]): unknown {
    const reply = recordOf(answer);
    const out = recordOf(reply?.out);
    const summary = recordOf(out?.Summary);
    return reply && out && summary?.Pending === false
        ? { ...reply, out: { ...out, Summary: { ...summary, Needs: list } } }
        : answer;
}

export function withNeeds(recording: Recording, list: readonly object[]): Recording {
    return {
        ...recording,
        exchanges: recording.exchanges.map((exchange) =>
            exchange.request.watch === 'overview.read' && exchange.request.target === undefined
                ? { ...exchange, answers: exchange.answers.map((one) => needed(one, list)) }
                : exchange,
        ),
    };
}
