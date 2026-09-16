import type { ReactNode } from 'react';

interface PageProps {
    children: ReactNode;
    className?: string;
}

/**
 * Unified page container: consistent padding and vertical spacing for all pages.
 * Use inside a route's page.tsx as the outermost wrapper.
 */
export function Page({ children, className = '' }: PageProps) {
    return (
        <div
            className={`flex min-h-full flex-col space-y-6 px-6 pt-6 pb-12 md:space-y-8 md:px-8 md:pt-8 ${className}`}
        >
            {children}
        </div>
    );
}
