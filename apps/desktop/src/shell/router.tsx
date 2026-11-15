import { createContext, useCallback, useContext, useEffect, useMemo, useReducer } from 'react';
import type { ReactNode } from 'react';

import { goBack, goForward, historyAt, navigate, replace } from './route';
import type { History, Route } from './route';

type Action =
    | { kind: 'back' }
    | { kind: 'forward' }
    | { kind: 'go'; to: Route }
    | { kind: 'replace'; to: Route };

function reduce(history: History, action: Action): History {
    switch (action.kind) {
        case 'go': {
            return navigate(history, action.to);
        }
        case 'replace': {
            return replace(history, action.to);
        }
        case 'back': {
            return goBack(history);
        }
        case 'forward': {
            return goForward(history);
        }
    }
}

export type Router = {
    route: Route;
    canGoBack: boolean;
    canGoForward: boolean;
    go: (to: Route) => void;
    replace: (to: Route) => void;
    back: () => void;
    forward: () => void;
};

const RouterContext = createContext<null | Router>(null);

type RouterProviderProps = {
    initial: Route;
    children: ReactNode;
    onRoute?: ((route: Route) => void) | undefined;
};

export function RouterProvider({ initial, children, onRoute }: RouterProviderProps) {
    const [history, dispatch] = useReducer(reduce, initial, historyAt);
    const go = useCallback((to: Route) => {
        dispatch({ kind: 'go', to });
    }, []);
    const replaced = useCallback((to: Route) => {
        dispatch({ kind: 'replace', to });
    }, []);
    const back = useCallback(() => {
        dispatch({ kind: 'back' });
    }, []);
    const forward = useCallback(() => {
        dispatch({ kind: 'forward' });
    }, []);
    const route = history.current;
    useEffect(() => {
        onRoute?.(route);
    }, [onRoute, route]);
    const canGoBack = history.back.length > 0;
    const canGoForward = history.forward.length > 0;
    const router = useMemo<Router>(
        () => ({ back, canGoBack, canGoForward, forward, go, replace: replaced, route }),
        [back, canGoBack, canGoForward, forward, go, replaced, route],
    );
    return <RouterContext value={router}>{children}</RouterContext>;
}

export function useRouter(): Router {
    const router = useContext(RouterContext);
    if (!router) {
        throw new Error('useRouter is read inside a RouterProvider.');
    }
    return router;
}
