import type { ReactNode } from 'react';

interface PageHeaderProps {
    title: string;
    description?: string;
    actions?: ReactNode;
    leading?: ReactNode;
}

/** Unified page header: optional leading glyph + large title + subtle description + optional right-aligned actions. Used across all top-level pages for consistency. */
export function PageHeader({ title, description, actions, leading }: PageHeaderProps) {
    return (
        <div className="flex items-start justify-between gap-4 overflow-visible">
            <div
                className={`flex min-w-0 items-center overflow-visible ${leading === undefined ? 'gap-3' : 'gap-5 md:gap-6'}`}
            >
                {leading}
                <div className="min-w-0">
                    <h1 className="font-heading text-foreground/90 text-2xl tracking-wide">
                        {title}
                    </h1>
                    {description !== undefined && description !== '' && (
                        <p className="text-muted-foreground/40 mt-1 text-xs">{description}</p>
                    )}
                </div>
            </div>
            {actions !== undefined && (
                <div className="flex shrink-0 items-center gap-2">{actions}</div>
            )}
        </div>
    );
}
