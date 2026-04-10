"use client";

import { useEffect, useState } from "react";
import {
  IconBrandDocker,
  IconRefresh,
  IconExternalLink,
  IconLoader2,
  IconAlertTriangle,
} from "@tabler/icons-react";
import { useDocker } from "@/contexts/docker-context";

/**
 * Full-area lock screen shown whenever the host Docker daemon is not
 * reachable. Replaces page content (the sidebar still renders, dimmed) so
 * users cannot trigger any action that depends on Docker.
 *
 * The component is intentionally calm: spwn IS a Docker control plane, so
 * "Docker offline" is the system state, not an emergency. We explain it,
 * keep retrying in the background, and recover automatically.
 */
export function DockerLockScreen() {
  const { status, lastChecked, refresh } = useDocker();
  const [refreshing, setRefreshing] = useState(false);
  const [secondsAgo, setSecondsAgo] = useState(0);

  // Tick the "last check" timer once a second so it feels alive.
  useEffect(() => {
    if (!lastChecked) return;
    const tick = () =>
      setSecondsAgo(Math.max(0, Math.floor((Date.now() - lastChecked) / 1000)));
    tick();
    const id = setInterval(tick, 1000);
    return () => clearInterval(id);
  }, [lastChecked]);

  const handleRetry = async () => {
    setRefreshing(true);
    try {
      await refresh();
    } finally {
      setRefreshing(false);
    }
  };

  // Two distinct failure modes get two distinct screens.
  const apiDown = status === undefined;
  const dockerDown = status && (!status.installed || !status.running);

  if (!apiDown && !dockerDown) return null;

  // ── API offline screen ──────────────────────────────────────────────
  if (apiDown) {
    return (
      <LockShell
        accent="amber"
        icon={<IconAlertTriangle size={28} className="text-amber-300" />}
        title="Can't reach the spwn API"
        subtitle="The desktop app couldn't talk to the local Go server."
        hint="Check that the spwn daemon is running. The app will reconnect automatically."
        primaryAction={
          <RetryButton onClick={handleRetry} loading={refreshing} />
        }
        secondsAgo={secondsAgo}
      />
    );
  }

  // ── Docker offline screen ──────────────────────────────────────────
  const installed = status?.installed ?? false;
  const installUrl =
    status?.platform === "linux"
      ? "https://docs.docker.com/engine/install/"
      : "https://www.docker.com/products/docker-desktop/";

  return (
    <LockShell
      accent="red"
      icon={<DockerPulse />}
      title={installed ? "Waiting for Docker" : "Docker isn't installed"}
      subtitle={
        installed
          ? "spwn needs the Docker daemon to be running. Start Docker Desktop and we'll pick it up automatically."
          : "Every spwn world runs inside a Docker container. Install Docker to continue."
      }
      hint={status?.hint}
      error={status?.error}
      primaryAction={
        <div className="flex flex-wrap items-center justify-center gap-2">
          {!installed && (
            <a
              href={installUrl}
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-1.5 rounded-lg border border-white/15 bg-white/[0.06] px-4 py-2 text-xs font-medium text-foreground/90 transition-colors hover:bg-white/[0.1]"
            >
              <IconBrandDocker size={14} />
              Install Docker
              <IconExternalLink size={11} />
            </a>
          )}
          <RetryButton onClick={handleRetry} loading={refreshing} />
        </div>
      }
      secondsAgo={secondsAgo}
    />
  );
}

// ── Sub-components ────────────────────────────────────────────────────

function LockShell({
  accent,
  icon,
  title,
  subtitle,
  hint,
  error,
  primaryAction,
  secondsAgo,
}: {
  accent: "red" | "amber";
  icon: React.ReactNode;
  title: string;
  subtitle: string;
  hint?: string;
  error?: string;
  primaryAction: React.ReactNode;
  secondsAgo: number;
}) {
  const ringColor =
    accent === "red"
      ? "from-red-500/20 via-transparent to-transparent"
      : "from-amber-500/20 via-transparent to-transparent";

  return (
    <div className="relative flex h-full w-full items-center justify-center px-6 py-10">
      {/* Ambient radial wash matching the accent */}
      <div
        aria-hidden
        className={`pointer-events-none absolute inset-0 bg-gradient-radial ${ringColor}`}
      />

      <div className="relative z-10 w-full max-w-md">
        <div className="rounded-2xl border border-white/[0.08] bg-black/30 px-8 py-9 text-center backdrop-blur-md shadow-2xl">
          <div className="mb-5 flex justify-center">{icon}</div>
          <h1 className="font-heading text-xl tracking-wide text-foreground/95">
            {title}
          </h1>
          <p className="mx-auto mt-2 max-w-sm text-[12px] leading-relaxed text-muted-foreground/70">
            {subtitle}
          </p>

          {(error || hint) && (
            <div className="mt-5 space-y-1.5 rounded-lg border border-white/[0.06] bg-white/[0.02] px-3 py-2.5 text-left">
              {error && (
                <p className="font-mono text-[10.5px] leading-snug text-muted-foreground/70 break-words">
                  {error}
                </p>
              )}
              {hint && (
                <p className="text-[11px] text-foreground/70">{hint}</p>
              )}
            </div>
          )}

          <div className="mt-6">{primaryAction}</div>

          <div className="mt-5 flex items-center justify-center gap-1.5 text-[10px] uppercase tracking-wider text-muted-foreground/40">
            <span className="relative flex h-1.5 w-1.5">
              <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-muted-foreground/40 opacity-75" />
              <span className="relative inline-flex h-1.5 w-1.5 rounded-full bg-muted-foreground/60" />
            </span>
            Checking every 3s · last check {secondsAgo}s ago
          </div>
        </div>
      </div>
    </div>
  );
}

function DockerPulse() {
  return (
    <div className="relative flex h-16 w-16 items-center justify-center">
      <span
        className="absolute inset-0 rounded-full border border-red-400/30"
        style={{ animation: "docker-pulse 2.4s ease-out infinite" }}
      />
      <span
        className="absolute inset-2 rounded-full border border-red-400/20"
        style={{ animation: "docker-pulse 2.4s ease-out infinite 0.6s" }}
      />
      <IconBrandDocker size={32} className="relative text-red-300/90" />
      <style jsx>{`
        @keyframes docker-pulse {
          0% {
            transform: scale(0.85);
            opacity: 0.9;
          }
          100% {
            transform: scale(1.6);
            opacity: 0;
          }
        }
      `}</style>
    </div>
  );
}

function RetryButton({
  onClick,
  loading,
}: {
  onClick: () => void;
  loading: boolean;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={loading}
      className="inline-flex items-center gap-1.5 rounded-lg border border-white/15 bg-white/[0.06] px-4 py-2 text-xs font-medium text-foreground/90 transition-colors hover:bg-white/[0.1] disabled:opacity-60"
    >
      {loading ? (
        <IconLoader2 size={13} className="animate-spin" />
      ) : (
        <IconRefresh size={13} />
      )}
      Retry now
    </button>
  );
}
