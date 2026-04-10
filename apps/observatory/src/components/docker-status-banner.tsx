"use client";

import { useEffect, useState } from "react";
import { IconAlertTriangle, IconExternalLink, IconRefresh } from "@tabler/icons-react";
import { apiGet } from "@/lib/api-client";

interface DockerStatus {
  installed: boolean;
  running: boolean;
  version?: string;
  error?: string;
  hint?: string;
  platform: string;
}

const POLL_INTERVAL_MS = 5000;

/**
 * Persistent banner shown across all pages whenever the host Docker daemon
 * is unreachable. Polls /api/system/docker every 5s and disappears as soon
 * as Docker is healthy again.
 *
 * This is the highest-priority diagnostic surface — without Docker, nothing
 * else in spwn works, and the previous behavior (silent failure) was the #1
 * complaint in QA.
 */
export function DockerStatusBanner() {
  const [status, setStatus] = useState<DockerStatus | null>(null);
  const [refreshing, setRefreshing] = useState(false);

  useEffect(() => {
    let cancelled = false;
    const fetchStatus = async () => {
      try {
        const data = await apiGet<DockerStatus>("/api/system/docker");
        if (!cancelled) setStatus(data);
      } catch {
        // If the API itself is down we can't tell — leave status as-is.
      }
    };
    fetchStatus();
    const id = setInterval(fetchStatus, POLL_INTERVAL_MS);
    return () => {
      cancelled = true;
      clearInterval(id);
    };
  }, []);

  // Hide while we don't yet know, and once Docker is healthy.
  if (!status || (status.installed && status.running)) return null;

  const handleRefresh = async () => {
    setRefreshing(true);
    try {
      const data = await apiGet<DockerStatus>("/api/system/docker");
      setStatus(data);
    } catch {
      /* ignore */
    } finally {
      setRefreshing(false);
    }
  };

  const title = !status.installed
    ? "Docker is not installed"
    : "Docker daemon is not running";
  const docsUrl = !status.installed
    ? status.platform === "darwin" || status.platform === "windows"
      ? "https://www.docker.com/products/docker-desktop/"
      : "https://docs.docker.com/engine/install/"
    : null;

  return (
    <div
      role="alert"
      className="relative z-[60] border-b border-red-500/30 bg-red-500/[0.08] backdrop-blur-sm"
    >
      <div className="mx-auto flex max-w-7xl items-start gap-3 px-4 py-2.5">
        <IconAlertTriangle
          size={16}
          className="mt-0.5 shrink-0 text-red-400"
        />
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-baseline gap-x-3 gap-y-0.5">
            <span className="text-[12px] font-medium text-red-200">
              {title}
            </span>
            {status.error && (
              <span className="truncate text-[11px] text-red-200/60">
                {status.error}
              </span>
            )}
          </div>
          {status.hint && (
            <p className="mt-0.5 text-[11px] text-red-200/70">{status.hint}</p>
          )}
        </div>
        <div className="flex shrink-0 items-center gap-1.5">
          {docsUrl && (
            <a
              href={docsUrl}
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-1 rounded-md border border-red-400/30 bg-red-500/10 px-2 py-1 text-[11px] font-medium text-red-100 transition-colors hover:bg-red-500/20"
            >
              Install
              <IconExternalLink size={11} />
            </a>
          )}
          <button
            onClick={handleRefresh}
            disabled={refreshing}
            className="inline-flex items-center gap-1 rounded-md border border-red-400/30 bg-red-500/10 px-2 py-1 text-[11px] font-medium text-red-100 transition-colors hover:bg-red-500/20 disabled:opacity-50"
          >
            <IconRefresh
              size={11}
              className={refreshing ? "animate-spin" : ""}
            />
            Retry
          </button>
        </div>
      </div>
    </div>
  );
}
