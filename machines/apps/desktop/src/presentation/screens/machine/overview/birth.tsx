import type { MachinesBirth } from '@jterrazz/station-api';

import { CodeBlock, Panel } from '../../../ui/index';
import { FAILED_CREATE_WORDS } from '../../../words/words';

export function FailedCreate({ birth }: { birth: MachinesBirth | null }) {
    if (birth?.failed !== true) {
        return null;
    }
    const lines = (birth.last_lines ?? []).map((text, index) => ({ number: index + 1, text }));
    return (
        <Panel note={FAILED_CREATE_WORDS.lines} title={birth.sentence}>
            <CodeBlock
                empty={FAILED_CREATE_WORDS.none}
                label={FAILED_CREATE_WORDS.lines}
                lines={lines}
                log
            />
        </Panel>
    );
}
