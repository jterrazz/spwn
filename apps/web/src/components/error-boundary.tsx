'use client';

import { IconAlertTriangle } from '@tabler/icons-react';
import React from 'react';

interface ErrorBoundaryProps {
    children: React.ReactNode;
    fallback?: React.ReactNode;
}

interface ErrorBoundaryState {
    hasError: boolean;
    error: Error | null;
}

export class ErrorBoundary extends React.Component<ErrorBoundaryProps, ErrorBoundaryState> {
    constructor(props: ErrorBoundaryProps) {
        super(props);
        this.state = { hasError: false, error: null };
    }

    static getDerivedStateFromError(error: Error): ErrorBoundaryState {
        return { hasError: true, error };
    }

    override render() {
        if (this.state.hasError) {
            if (this.props.fallback) {
                return this.props.fallback;
            }

            return (
                <div className="flex flex-col items-center justify-center px-8 py-16">
                    <div className="mb-4 flex h-14 w-14 items-center justify-center rounded-2xl border border-red-500/20 bg-red-500/10">
                        <IconAlertTriangle className="text-red-400" size={24} />
                    </div>
                    <h2 className="font-heading text-foreground/80 mb-2 text-lg">
                        Something went wrong
                    </h2>
                    <p className="text-muted-foreground/40 mb-4 max-w-md text-center font-mono text-sm">
                        {this.state.error?.message || 'An unexpected error occurred'}
                    </p>
                    <button
                        className="text-foreground/60 hover:text-foreground/80 rounded-xl border border-white/[0.06] bg-white/[0.04] px-4 py-2 text-sm transition-all hover:bg-white/[0.08]"
                        // oxlint-disable-next-line react/no-set-state -- class-based ErrorBoundary has no hook alternative for resetting its own state
                        onClick={() => this.setState({ hasError: false, error: null })}
                    >
                        Try again
                    </button>
                </div>
            );
        }

        return this.props.children;
    }
}
