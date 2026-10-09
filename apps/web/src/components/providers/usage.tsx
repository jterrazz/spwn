import type { ProviderUsage } from './meta';

function barTone(percent: number): string {
    if (percent > 90) {
        return 'bg-red-400';
    }
    return percent > 70 ? 'bg-amber-400' : 'bg-green-400/70';
}

function UsageBar({
    label,
    used,
    limit,
    currency = '',
}: {
    label: string;
    used: number;
    limit: number;
    currency?: string;
}) {
    const percent = limit > 0 ? Math.min((used / limit) * 100, 100) : 0;
    return (
        <div className="space-y-1">
            <div className="flex items-center justify-between">
                <span className="text-muted-foreground/40 text-[10px]">{label}</span>
                <span className="text-muted-foreground/50 font-mono text-[10px]">
                    {currency === ''
                        ? `${percent.toFixed(0)}%`
                        : `${currency}${used.toFixed(2)} / ${currency}${limit.toFixed(2)}`}
                </span>
            </div>
            <div className="h-1 overflow-hidden rounded-full bg-white/[0.06]">
                <div
                    className={`h-full rounded-full transition-all duration-500 ${barTone(percent)}`}
                    style={{ width: `${percent}%` }}
                />
            </div>
        </div>
    );
}

/** What a provider's plan has used: its session, its week, its credits. */
export function Usage({ usage }: { usage: ProviderUsage }) {
    return (
        <div className="ml-13 max-w-xs space-y-2">
            {usage.session !== undefined && (
                <UsageBar
                    label={`Session (${usage.session.label})`}
                    limit={usage.session.limit}
                    used={usage.session.used}
                />
            )}
            {usage.weekly !== undefined && (
                <UsageBar
                    label={`Weekly (${usage.weekly.label})`}
                    limit={usage.weekly.limit}
                    used={usage.weekly.used}
                />
            )}
            {usage.credits !== undefined && (
                <UsageBar
                    currency={usage.credits.currency}
                    label="Credits"
                    limit={usage.credits.limit}
                    used={usage.credits.used}
                />
            )}
        </div>
    );
}
