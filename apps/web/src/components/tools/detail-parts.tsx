'use client';

export function MetaCard({
    label,
    value,
    icon,
}: {
    label: string;
    value: string;
    icon?: React.ReactNode;
}) {
    return (
        <div className="rounded-lg border border-white/[0.06] bg-white/[0.02] px-4 py-3">
            <p className="text-muted-foreground/25 mb-1 text-[9px] tracking-widest uppercase">
                {label}
            </p>
            <div className="flex items-center gap-1.5">
                {icon}
                <span className="text-foreground/70 font-mono text-sm">{value}</span>
            </div>
        </div>
    );
}

export function DetailSection({ label, children }: { label: string; children: React.ReactNode }) {
    return (
        <div>
            <p className="text-muted-foreground/25 mb-2 text-[10px] tracking-widest uppercase">
                {label}
            </p>
            {children}
        </div>
    );
}
