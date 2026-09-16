/**
 * A thin animated shimmer bar + progress message for long-running operations.
 *
 * Usage:
 *   <ProgressShimmer active={isLoading} message="Building image..." />
 */
export function ProgressShimmer({
    active,
    message,
    className,
}: {
    active: boolean;
    message?: string;
    className?: string;
}) {
    if (!active) {
        return null;
    }
    return (
        <div className={`w-full space-y-1.5 ${className ?? ''}`}>
            {/* Shimmer bar */}
            <div className="h-0.5 w-full overflow-hidden rounded-full bg-white/[0.04]">
                <div
                    className="h-full w-1/3 rounded-full bg-emerald-500/30"
                    style={{
                        animation: 'progressSlide 1.5s ease-in-out infinite',
                    }}
                />
            </div>
            {/* Message */}
            {message && (
                <p className="text-muted-foreground/40 animate-pulse font-mono text-[11px]">
                    {message}
                </p>
            )}
            <style>{`
        @keyframes progressSlide {
          0% { transform: translateX(-100%); }
          50% { transform: translateX(200%); }
          100% { transform: translateX(200%); }
        }
      `}</style>
        </div>
    );
}
