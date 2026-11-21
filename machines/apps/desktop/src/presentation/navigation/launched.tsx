import { useEffect } from 'react';

import { holdsData } from '../../application/use-cases/launch';
import { useReach, useWatch } from '../dependencies/dependencies';

// invariant: the frame's own machines.list, the same request, so watching it here asks the engine nothing more; the
// invariant: effect runs after the commit that drew it.
export function Launched({ onDrawn }: { onDrawn: () => void }) {
    const reach = useReach();
    const machines = useWatch('machines.list', {});
    const holds = holdsData(reach, machines);
    useEffect(() => {
        if (holds) {
            onDrawn();
        }
    }, [holds, onDrawn]);
    return null;
}
