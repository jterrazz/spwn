import Link from 'next/link';

export default function NotFound() {
    return (
        <div className="flex min-h-screen flex-col items-center justify-center">
            <div className="text-center">
                <h1 className="font-heading text-foreground/20 mb-4 text-6xl tracking-wide">404</h1>
                <p className="text-muted-foreground/40 font-heading mb-2 text-lg">Page not found</p>
                <p className="text-muted-foreground/25 mb-8 font-mono text-sm">
                    This corner of the universe doesn&apos;t exist yet.
                </p>
                <Link
                    className="text-foreground/60 hover:text-foreground/80 inline-flex items-center gap-2 rounded-xl border border-white/[0.06] bg-white/[0.04] px-5 py-2.5 text-sm transition-all hover:bg-white/[0.08]"
                    href="/"
                >
                    ← Back to home
                </Link>
            </div>
        </div>
    );
}
