"use client";

import { useState, useEffect, useCallback } from "react";
import { apiGet } from "@/lib/api-client";

const VERSION_CHECK_INTERVAL = 5 * 60 * 1000; // 5 minutes

export interface VersionInfo {
  current: string;
  latest: string;
  updateAvailable: boolean;
  releaseUrl: string;
}

export function useVersion() {
  const [version, setVersion] = useState<VersionInfo | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchVersion = useCallback(async () => {
    try {
      const data = await apiGet<VersionInfo>("/api/version");
      setVersion(data);
    } catch {
      // silently fail
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchVersion();
    const interval = setInterval(fetchVersion, VERSION_CHECK_INTERVAL);
    return () => clearInterval(interval);
  }, [fetchVersion]);

  return { version, loading };
}
