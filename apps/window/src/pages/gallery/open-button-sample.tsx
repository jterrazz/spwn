import { Client } from '../../engine/client';
import { NO_OPENER } from '../../engine/opener';
import { EngineProvider } from '../../engine/react';
import { replayTransport } from '../../engine/replay';
import { OpenButton } from '../kit/opens';
import type { OpenButtonProps } from '../kit/opens';

const NO_ENGINE = new Client(
    replayTransport({ exchanges: [], station: 'gallery', world: 'gallery' }),
);

export function OpenButtonSample(props: OpenButtonProps) {
    return (
        <EngineProvider client={NO_ENGINE} opener={NO_OPENER}>
            <OpenButton {...props} />
        </EngineProvider>
    );
}
