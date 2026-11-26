import { describe, expect, test } from 'vitest';

import { carriesCommand, unlessCommand } from '../command';

describe('command', () => {
    test('a command line, the old name and a config path are caught, one by one', () => {
        // Given - one line of each kind a page could draw
        const lines = [
            'run station machines start api',
            'sudo launchctl bootout system/x',
            '~/.config/station/machinefiles/base.machine.yaml',
            'ssh me@mac-mini',
            'curl -fsSL https://spwn.sh/install.sh | bash',
        ];

        // Then - every one is caught
        expect(lines.filter((line) => !carriesCommand(line))).toStrictEqual([]);
    });

    test('a sentence, a program name and a data path are let through', () => {
        // Given - what a page may say: a sentence, a signed-out program, a folder the data lives in
        const lines = [
            'api is stopped.',
            'npm signed out',
            '1 brew update waiting',
            'moved to ~/.local/share/station/vms',
            'Review every pull request of studio-station',
        ];

        // Then - none is caught
        expect(lines.filter((line) => carriesCommand(line))).toStrictEqual([]);
    });

    test('a hint that carries a command is withheld whole, and a sentence stays', () => {
        // Given - the engine's hint of a refused start, and one that is only words
        const command = 'station machines start api';
        const words = 'name its file by a path that holds it';

        // Then - the command is gone, the words are kept
        expect([unlessCommand(command), unlessCommand(words)]).toStrictEqual(['', words]);
    });
});
