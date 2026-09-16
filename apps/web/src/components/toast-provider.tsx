'use client';

import { createContext, useCallback, useContext, useMemo, useState } from 'react';
import type { ReactNode } from 'react';

interface Toast {
    id: number;
    message: string;
    type: 'error' | 'info' | 'success';
}

interface ToastContextValue {
    toast: (message: string, type?: Toast['type']) => void;
}

const ToastContext = createContext<ToastContextValue>({
    toast: () => {},
});

export function useToast() {
    return useContext(ToastContext);
}

let toastId = 0;

function toastClass(type: Toast['type']): string {
    if (type === 'success') {
        return 'bg-green-500/10 border-green-500/20 text-green-400';
    }
    if (type === 'error') {
        return 'bg-red-500/10 border-red-500/20 text-red-400';
    }
    return 'bg-blue-500/10 border-blue-500/20 text-blue-400';
}

export function ToastProvider({ children }: { children: ReactNode }) {
    const [toasts, setToasts] = useState<Toast[]>([]);

    const toast = useCallback((message: string, type: Toast['type'] = 'success') => {
        const id = ++toastId;
        setToasts((prev) => [...prev, { id, message, type }]);
        setTimeout(() => {
            setToasts((prev) => prev.filter((t) => t.id !== id));
        }, 3000);
    }, []);

    const contextValue = useMemo(() => ({ toast }), [toast]);

    return (
        <ToastContext.Provider value={contextValue}>
            {children}
            {/* Toast container */}
            <div className="pointer-events-none fixed right-6 bottom-6 z-50 flex flex-col gap-2">
                {toasts.map((t) => (
                    <div
                        className={`animate-in fade-in slide-in-from-bottom-2 pointer-events-auto rounded-lg border px-4 py-2.5 font-mono text-xs backdrop-blur-sm duration-200 ${toastClass(t.type)}`}
                        key={t.id}
                    >
                        {t.message}
                    </div>
                ))}
            </div>
        </ToastContext.Provider>
    );
}
