import { EngineProvider } from '../../engine/react';
import { OpenButton } from '../kit/opens';
import type { OpenButtonProps } from '../kit/opens';
import { useSample } from './sample';

export function OpenButtonSample(props: OpenButtonProps) {
    return (
        <EngineProvider dependencies={useSample()}>
            <OpenButton {...props} />
        </EngineProvider>
    );
}
