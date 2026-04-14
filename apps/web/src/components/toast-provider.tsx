'use client';

import { createContext, type ReactNode, useCallback, useContext, useState } from 'react';

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

export function ToastProvider({ children }: { children: ReactNode }) {
    const [toasts, setToasts] = useState<Toast[]>([]);

    const toast = useCallback((message: string, type: Toast['type'] = 'success') => {
        const id = ++toastId;
        setToasts((prev) => [...prev, { id, message, type }]);
        setTimeout(() => {
            setToasts((prev) => prev.filter((t) => t.id !== id));
        }, 3000);
    }, []);

    return (
        <ToastContext.Provider value={{ toast }}>
            {children}
            {/* Toast container */}
            <div className="fixed bottom-6 right-6 z-50 flex flex-col gap-2 pointer-events-none">
                {toasts.map((t) => (
                    <div
                        className={`pointer-events-auto px-4 py-2.5 rounded-lg text-xs font-mono border backdrop-blur-sm animate-in fade-in slide-in-from-bottom-2 duration-200 ${
                            t.type === 'success'
                                ? 'bg-green-500/10 border-green-500/20 text-green-400'
                                : t.type === 'error'
                                  ? 'bg-red-500/10 border-red-500/20 text-red-400'
                                  : 'bg-blue-500/10 border-blue-500/20 text-blue-400'
                        }`}
                        key={t.id}
                    >
                        {t.message}
                    </div>
                ))}
            </div>
        </ToastContext.Provider>
    );
}
