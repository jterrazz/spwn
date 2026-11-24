import { useEffect } from 'react';

import type { Found } from '../../domain/machines/machines';
import { settledRoute } from './route';
import type { Router } from './router';

export function useSettledRoute({ route, replace }: Router, found: Found | undefined): void {
    const settled = settledRoute(route, found !== undefined && found.vm === undefined);
    useEffect(() => {
        if (settled !== route) {
            replace(settled);
        }
    }, [replace, route, settled]);
}
