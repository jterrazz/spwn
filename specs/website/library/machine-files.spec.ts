import { describe, test } from 'vitest';

import '../website.specification.js';

const FILES = 'every machine file with its source and whether it is a mixin, on one read';

describe('machine files', () => {
    test.todo('D1 waits for the machines that follow a file, and whether each matches it');

    test.todo('D2 lot 5');

    test.todo('D3 lot 5');

    test.todo('D7 lot 6');

    test.todo("D11 waits for the file's description and its VM's resources on machinefile.explain");

    test.todo('D12 lot 6');

    test.todo(`D18 waits for ${FILES}`);

    test.todo('D19 lot 6');

    test.todo(
        "D20 waits for the local file's additions, removals and values on machinefile.explain",
    );

    test.todo(`D21 waits for ${FILES}`);

    test.todo("D23 waits for a call that opens the owner's file to edit");

    test.todo("D24 waits for the file's text on machinefile.explain");
});
