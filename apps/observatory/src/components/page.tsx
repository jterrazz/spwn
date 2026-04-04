import type { ReactNode } from "react";

interface PageProps {
  children: ReactNode;
  className?: string;
}

/**
 * Unified page container: consistent padding and vertical spacing for all pages.
 * Use inside a route's page.tsx as the outermost wrapper.
 */
export function Page({ children, className = "" }: PageProps) {
  return (
    <div className={`px-6 md:px-8 pt-6 md:pt-8 pb-12 space-y-6 md:space-y-8 ${className}`}>
      {children}
    </div>
  );
}
