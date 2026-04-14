"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";
import { apiGet } from "@/lib/api-client";

export interface DockerStatus {
  installed: boolean;
  running: boolean;
  version?: string;
  error?: string;
  hint?: string;
  platform: string;
}

interface DockerContextValue {
  /** Latest probe result. null = haven't loaded yet, undefined = API offline. */
  status: DockerStatus | null | undefined;
  /** Wall-clock timestamp of the last successful probe. */
  lastChecked: number | null;
  /** Force an immediate refresh. */
  refresh: () => Promise<void>;
  /** Convenience: true once we know Docker is fully usable. */
  ready: boolean;
}

const DockerContext = createContext<DockerContextValue>({
  status: null,
  lastChecked: null,
  refresh: async () => {},
  ready: false,
});

const POLL_INTERVAL_MS = 3000;

/**
 * Single source of truth for Docker daemon health. Polls the Go API every
 * 3 seconds and exposes the latest probe result through React context.
 *
 * Status semantics:
 *   null       - first load, we don't know yet (render skeleton, not lock)
 *   undefined  - the API itself is unreachable (different failure mode)
 *   object     - actual probe result; check `installed && running`
 */
export function DockerProvider({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = useState<DockerStatus | null | undefined>(null);
  const [lastChecked, setLastChecked] = useState<number | null>(null);
  const mounted = useRef(true);

  const refresh = useCallback(async () => {
    try {
      const data = await apiGet<DockerStatus>("/api/system/docker");
      if (!mounted.current) return;
      setStatus(data);
      setLastChecked(Date.now());
    } catch {
      if (!mounted.current) return;
      // The API itself didn't answer - distinct from "Docker is down".
      setStatus(undefined);
      setLastChecked(Date.now());
    }
  }, []);

  useEffect(() => {
    mounted.current = true;
    // refresh() is async - setState only fires inside the resolved promise,
    // never synchronously, so this does not cause a cascading render.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void refresh();
    const id = setInterval(() => {
      void refresh();
    }, POLL_INTERVAL_MS);
    return () => {
      mounted.current = false;
      clearInterval(id);
    };
  }, [refresh]);

  const ready = !!(status && status.installed && status.running);

  return (
    <DockerContext.Provider value={{ status, lastChecked, refresh, ready }}>
      {children}
    </DockerContext.Provider>
  );
}

export function useDocker(): DockerContextValue {
  return useContext(DockerContext);
}
