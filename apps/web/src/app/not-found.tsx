import Link from "next/link";

export default function NotFound() {
  return (
    <div className="flex flex-col items-center justify-center min-h-screen">
      <div className="text-center">
        <h1 className="text-6xl font-heading tracking-wide text-foreground/20 mb-4">
          404
        </h1>
        <p className="text-lg text-muted-foreground/40 font-heading mb-2">
          Page not found
        </p>
        <p className="text-sm text-muted-foreground/25 font-mono mb-8">
          This corner of the universe doesn&apos;t exist yet.
        </p>
        <Link
          href="/"
          className="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl text-sm bg-white/[0.04] text-foreground/60 hover:text-foreground/80 hover:bg-white/[0.08] border border-white/[0.06] transition-all"
        >
          ← Back to home
        </Link>
      </div>
    </div>
  );
}
