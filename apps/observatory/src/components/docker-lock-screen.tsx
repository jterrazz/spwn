"use client";

import { useEffect, useState } from "react";
import {
  IconBrandDocker,
  IconRefresh,
  IconExternalLink,
  IconLoader2,
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

  // Three distinct states all get the same Docker-themed lock screen,
  // because for users the actionable next step is the same in every
  // case: make sure Docker is running. The "API offline" branding the
  // previous version showed was technically accurate but useless — most
  // users don't know what the spwn API is, only that Docker needs to
  // be on for any of this to work. We surface API status as a quiet
  // diagnostic line at the bottom instead.

  const apiDown = status === undefined;
  const dockerDown = !!(status && (!status.installed || !status.running));

  if (!apiDown && !dockerDown) return null;

  const installed = status?.installed ?? false;
  // When the API is unreachable we don't actually know whether Docker is
  // installed, so default to the "running but offline" copy — that's the
  // most common cause and the install link would be wrong otherwise.
  const showInstallCta = !apiDown && !installed;
  const installUrl =
    status?.platform === "linux"
      ? "https://docs.docker.com/engine/install/"
      : "https://www.docker.com/products/docker-desktop/";

  // Title + subtitle pick the most accurate copy we can given what we
  // know. All three branches use the same Docker pulse glyph.
  let title: string;
  let subtitle: string;
  if (apiDown) {
    title = "Connecting to spwn";
    subtitle =
      "The desktop app is waiting for the local spwn daemon. Make sure Docker is running — the daemon will come up with it.";
  } else if (!installed) {
    title = "Docker isn't installed";
    subtitle =
      "Every spwn world runs inside a Docker container. Install Docker to continue.";
  } else {
    title = "Waiting for Docker";
    subtitle =
      "spwn needs the Docker daemon to be running. Start Docker Desktop and we'll pick it up automatically.";
  }

  return (
    <LockShell
      icon={<DockerPulse />}
      title={title}
      subtitle={subtitle}
      hint={status?.hint}
      error={status?.error}
      diagnostic={apiDown ? "spwn API isn't responding" : undefined}
      primaryAction={
        <div className="flex flex-wrap items-center justify-center gap-2">
          {showInstallCta && (
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
  icon,
  title,
  subtitle,
  hint,
  error,
  diagnostic,
  primaryAction,
  secondsAgo,
}: {
  icon: React.ReactNode;
  title: string;
  subtitle: string;
  hint?: string;
  error?: string;
  /** Optional dim line shown above the polling footer for sub-failures
   *  (e.g. "spwn API isn't responding") that aren't the primary cause. */
  diagnostic?: string;
  primaryAction: React.ReactNode;
  secondsAgo: number;
}) {
  return (
    <div className="relative flex h-full w-full items-center justify-center px-6 py-10">
      {/* Ambient radial wash matching the Docker accent */}
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 bg-gradient-radial from-red-500/20 via-transparent to-transparent"
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

          {diagnostic && (
            <p className="mt-4 text-[10px] uppercase tracking-wider text-muted-foreground/40">
              {diagnostic}
            </p>
          )}

          <div className="mt-3 flex items-center justify-center gap-1.5 text-[10px] uppercase tracking-wider text-muted-foreground/40">
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
