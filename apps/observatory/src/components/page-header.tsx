import type { ReactNode } from "react";

interface PageHeaderProps {
  title: string;
  description?: string;
  actions?: ReactNode;
  /** Optional leading element (e.g. a world planet) rendered before the title. */
  leading?: ReactNode;
}

/**
 * Unified page header: optional leading glyph + large title + subtle
 * description + optional right-aligned actions. Used across all top-level
 * pages for consistency.
 */
export function PageHeader({ title, description, actions, leading }: PageHeaderProps) {
  return (
    <div className="flex items-start justify-between gap-4 overflow-visible">
      <div className={`flex items-center min-w-0 overflow-visible ${leading ? "gap-5 md:gap-6" : "gap-3"}`}>
        {leading}
        <div className="min-w-0">
          <h1 className="text-2xl font-heading tracking-wide text-foreground/90">{title}</h1>
          {description && (
            <p className="text-xs text-muted-foreground/40 mt-1">{description}</p>
          )}
        </div>
      </div>
      {actions && <div className="flex items-center gap-2 shrink-0">{actions}</div>}
    </div>
  );
}
