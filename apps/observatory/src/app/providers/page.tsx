"use client";

import { useState, useEffect } from "react";
import { Skeleton } from "@/components/ui/skeleton";
import { goApiUrl } from "@/lib/api-client";
import { usePageTitle } from "@/hooks/use-page-title";
import {
  IconKey,
  IconPlugConnected,
  IconRefresh,
  IconCheck,
  IconX,
  IconAlertTriangle,
  IconExternalLink,
  IconEye,
  IconEyeOff,
} from "@tabler/icons-react";

interface ProviderUsage {
  session?: { used: number; limit: number; label: string };
  weekly?: { used: number; limit: number; label: string };
  credits?: { used: number; limit: number; currency: string };
}

interface ProviderInfo {
  provider: string;
  connected: boolean;
  credentialType: string | null;
  source: string | null;
  error: string | null;
  plan: string | null;
  usage: ProviderUsage | null;
}

const PROVIDER_META: Record<
  string,
  { name: string; icon: string; color: string; docsUrl: string; envKey: string }
> = {
  anthropic: {
    name: "Anthropic",
    icon: "◆",
    color: "text-orange-400",
    docsUrl: "https://console.anthropic.com/settings/keys",
    envKey: "ANTHROPIC_API_KEY",
  },
  openai: {
    name: "OpenAI",
    icon: "◎",
    color: "text-green-400",
    docsUrl: "https://platform.openai.com/api-keys",
    envKey: "OPENAI_API_KEY",
  },
  google: {
    name: "Google",
    icon: "◈",
    color: "text-blue-400",
    docsUrl: "https://aistudio.google.com/app/apikey",
    envKey: "GOOGLE_API_KEY",
  },
};

function StatusDot({ status }: { status: "connected" | "error" | "unconfigured" }) {
  if (status === "connected") {
    return (
      <span className="relative flex h-2.5 w-2.5">
        <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400/40" />
        <span className="relative inline-flex rounded-full h-2.5 w-2.5 bg-green-400" />
      </span>
    );
  }
  if (status === "error") {
    return <span className="w-2.5 h-2.5 rounded-full bg-red-400" />;
  }
  return <span className="w-2.5 h-2.5 rounded-full bg-white/15" />;
}

function UsageBar({
  label,
  used,
  limit,
  suffix,
}: {
  label: string;
  used: number;
  limit: number;
  suffix?: string;
}) {
  const pct = limit > 0 ? Math.min((used / limit) * 100, 100) : 0;
  const barColor =
    pct > 90
      ? "bg-red-400"
      : pct > 70
        ? "bg-amber-400"
        : "bg-green-400/70";

  return (
    <div className="space-y-1">
      <div className="flex items-center justify-between">
        <span className="text-[10px] text-muted-foreground/40">{label}</span>
        <span className="text-[10px] font-mono text-muted-foreground/50">
          {suffix
            ? `${suffix}${used.toFixed(2)} / ${suffix}${limit.toFixed(2)}`
            : `${pct.toFixed(0)}%`}
        </span>
      </div>
      <div className="h-1.5 rounded-full bg-white/[0.06] overflow-hidden">
        <div
          className={`h-full rounded-full transition-all duration-500 ${barColor}`}
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  );
}

function CredentialBadge({ type }: { type: string | null }) {
  if (!type) return null;
  const colors: Record<string, string> = {
    "API Key": "bg-blue-500/15 text-blue-400/90 border-blue-500/20",
    OAuth: "bg-purple-500/15 text-purple-400/90 border-purple-500/20",
    Keychain: "bg-amber-500/15 text-amber-400/90 border-amber-500/20",
  };
  return (
    <span
      className={`text-[9px] font-mono uppercase tracking-wider px-1.5 py-0.5 rounded-full border ${colors[type] ?? "bg-white/10 text-white/60 border-white/10"}`}
    >
      {type}
    </span>
  );
}

function PlanBadge({ plan }: { plan: string | null }) {
  if (!plan) return null;
  const colors: Record<string, string> = {
    Pro: "bg-green-500/15 text-green-400/90 border-green-500/20",
    Free: "bg-white/10 text-white/50 border-white/10",
    "API Key": "bg-blue-500/15 text-blue-400/90 border-blue-500/20",
  };
  return (
    <span
      className={`text-[9px] font-mono uppercase tracking-wider px-1.5 py-0.5 rounded-full border ${colors[plan] ?? "bg-white/10 text-white/50 border-white/10"}`}
    >
      {plan}
    </span>
  );
}

function ConfigureModal({
  provider,
  onClose,
  onSave,
}: {
  provider: string;
  onClose: () => void;
  onSave: (token: string) => void;
}) {
  const [token, setToken] = useState("");
  const [saving, setSaving] = useState(false);
  const [showToken, setShowToken] = useState(false);
  const meta = PROVIDER_META[provider];

  const handleSave = async () => {
    if (!token.trim()) return;
    setSaving(true);
    onSave(token.trim());
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div
        className="absolute inset-0 bg-black/40 backdrop-blur-sm"
        onClick={onClose}
      />
      <div className="relative z-10 w-full max-w-md mx-4 rounded-2xl bg-popover/95 backdrop-blur-md border border-white/[0.08] shadow-2xl p-6">
        <div className="flex items-center justify-between mb-5">
          <h3 className="text-sm font-heading tracking-wide text-foreground/80">
            Configure {meta?.name ?? provider}
          </h3>
          <button
            onClick={onClose}
            className="p-1 text-muted-foreground/30 hover:text-muted-foreground/60 transition-colors"
          >
            <IconX size={16} />
          </button>
        </div>

        <div className="space-y-4">
          <div>
            <label className="text-[10px] uppercase tracking-widest text-muted-foreground/40 block mb-2">
              API Key
            </label>
            <div className="relative">
              <input
                type={showToken ? "text" : "password"}
                value={token}
                onChange={(e) => setToken(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") handleSave();
                }}
                placeholder={`sk-... or paste your ${meta?.name ?? provider} API key`}
                className="w-full bg-white/[0.03] border border-white/[0.08] rounded-lg px-4 py-3 pr-10 text-sm font-mono text-foreground/80 placeholder:text-muted-foreground/25 focus:outline-none focus:border-white/[0.15] transition-colors"
                autoFocus
              />
              <button
                type="button"
                onClick={() => setShowToken(!showToken)}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground/30 hover:text-muted-foreground/60 transition-colors"
              >
                {showToken ? <IconEyeOff size={16} /> : <IconEye size={16} />}
              </button>
            </div>
            <div className="flex items-center gap-2 mt-2">
              <p className="text-[10px] text-muted-foreground/25">
                Saved via <span className="font-mono">spwn auth token</span>
              </p>
              {meta?.docsUrl && (
                <a
                  href={meta.docsUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-[10px] text-blue-400/50 hover:text-blue-400/80 transition-colors flex items-center gap-0.5"
                >
                  Get key <IconExternalLink size={10} />
                </a>
              )}
            </div>
          </div>

          <div className="flex gap-3">
            <button
              onClick={onClose}
              className="flex-1 px-4 py-2.5 rounded-xl text-sm text-muted-foreground/50 hover:text-foreground/70 hover:bg-white/[0.04] transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={handleSave}
              disabled={!token.trim() || saving}
              className="flex-1 px-4 py-2.5 rounded-xl text-sm bg-white/[0.06] text-foreground/70 hover:bg-white/[0.1] border border-white/[0.08] transition-all disabled:opacity-30 disabled:cursor-not-allowed"
            >
              {saving ? (
                <span className="flex items-center justify-center gap-2">
                  <span className="w-3.5 h-3.5 border-2 border-foreground/30 border-t-foreground/70 rounded-full animate-spin" />
                  Saving...
                </span>
              ) : (
                "Save"
              )}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

function ProviderCard({
  provider,
  onCheckConnection,
  onConfigure,
  checking,
}: {
  provider: ProviderInfo;
  onCheckConnection: () => void;
  onConfigure: () => void;
  checking: boolean;
}) {
  const meta = PROVIDER_META[provider.provider] ?? {
    name: provider.provider,
    icon: "●",
    color: "text-white/60",
    docsUrl: "",
    envKey: "",
  };

  const status = provider.connected
    ? "connected"
    : provider.error
      ? "error"
      : "unconfigured";

  return (
    <div className="glass-subtle rounded-xl p-5 space-y-4">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-3">
          <div
            className={`w-10 h-10 rounded-xl bg-white/[0.04] border border-white/[0.06] flex items-center justify-center text-lg ${meta.color}`}
          >
            {meta.icon}
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h3 className="text-sm font-heading tracking-wide text-foreground/80">
                {meta.name}
              </h3>
              <StatusDot status={status} />
            </div>
            <p className="text-[10px] font-mono text-muted-foreground/30 mt-0.5">
              {status === "connected"
                ? "Connected"
                : status === "error"
                  ? "Error"
                  : "Not configured"}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-1.5">
          <CredentialBadge type={provider.credentialType} />
          <PlanBadge plan={provider.plan} />
        </div>
      </div>

      {/* Source */}
      {provider.source && (
        <div className="flex items-center gap-2">
          <IconKey size={12} className="text-muted-foreground/20" />
          <span className="text-[11px] font-mono text-muted-foreground/35">
            {provider.source}
          </span>
        </div>
      )}

      {/* Usage bars */}
      {provider.usage && (
        <div className="space-y-2.5 pt-1">
          {provider.usage.session && (
            <UsageBar
              label={`Session (${provider.usage.session.label})`}
              used={provider.usage.session.used}
              limit={provider.usage.session.limit}
            />
          )}
          {provider.usage.weekly && (
            <UsageBar
              label={`Weekly (${provider.usage.weekly.label})`}
              used={provider.usage.weekly.used}
              limit={provider.usage.weekly.limit}
            />
          )}
          {provider.usage.credits && (
            <UsageBar
              label="Credits"
              used={provider.usage.credits.used}
              limit={provider.usage.credits.limit}
              suffix={provider.usage.credits.currency}
            />
          )}
        </div>
      )}

      {/* Error */}
      {provider.error && (
        <div className="rounded-lg bg-red-500/10 border border-red-500/20 px-3 py-2 flex items-start gap-2">
          <IconAlertTriangle
            size={14}
            className="text-red-400/70 mt-0.5 shrink-0"
          />
          <p className="text-[11px] text-red-400/80 font-mono leading-relaxed">
            {provider.error}
          </p>
        </div>
      )}

      {/* Actions */}
      <div className="flex items-center gap-2 pt-1">
        <button
          onClick={onConfigure}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-[11px] bg-white/[0.04] text-foreground/50 hover:text-foreground/70 hover:bg-white/[0.08] border border-white/[0.06] transition-all"
        >
          <IconKey size={12} />
          Configure
        </button>
        <button
          onClick={onCheckConnection}
          disabled={checking}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-[11px] bg-white/[0.04] text-foreground/50 hover:text-foreground/70 hover:bg-white/[0.08] border border-white/[0.06] transition-all disabled:opacity-40"
        >
          {checking ? (
            <span className="w-3 h-3 border-2 border-foreground/30 border-t-foreground/70 rounded-full animate-spin" />
          ) : (
            <IconRefresh size={12} />
          )}
          {checking ? "Checking..." : "Check Connection"}
        </button>
      </div>
    </div>
  );
}

function ProviderCardSkeleton() {
  return (
    <div className="glass-subtle rounded-xl p-5 space-y-4">
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-3">
          <Skeleton className="w-10 h-10 rounded-xl" />
          <div>
            <Skeleton className="h-4 w-24" />
            <Skeleton className="h-2.5 w-16 mt-1.5" />
          </div>
        </div>
        <Skeleton className="h-4 w-14 rounded-full" />
      </div>
      <Skeleton className="h-3 w-40" />
      <div className="space-y-2">
        <Skeleton className="h-6 w-full" />
        <Skeleton className="h-6 w-full" />
      </div>
      <div className="flex gap-2">
        <Skeleton className="h-7 w-24 rounded-lg" />
        <Skeleton className="h-7 w-32 rounded-lg" />
      </div>
    </div>
  );
}

export default function ProvidersPage() {
  usePageTitle("Settings");
  const [providers, setProviders] = useState<ProviderInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [checking, setChecking] = useState<string | null>(null);
  const [configuring, setConfiguring] = useState<string | null>(null);
  const [feedback, setFeedback] = useState<{
    message: string;
    type: "success" | "error";
  } | null>(null);

  const showFeedback = (message: string, type: "success" | "error") => {
    setFeedback({ message, type });
    setTimeout(() => setFeedback(null), 3000);
  };

  const fetchProviders = async () => {
    try {
      const res = await fetch(goApiUrl("/api/auth/providers"));
      if (!res.ok) throw new Error("Failed to fetch providers");
      const data = await res.json();
      setProviders(data.providers ?? []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load providers");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchProviders();
  }, []);

  const handleCheck = async (providerName: string) => {
    setChecking(providerName);
    try {
      const res = await fetch(goApiUrl("/api/auth/check"), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ provider: providerName }),
      });
      const data = await res.json();
      if (data.connected) {
        showFeedback(`${providerName} connection verified`, "success");
        // Update the provider in state
        setProviders((prev) =>
          prev.map((p) =>
            p.provider === providerName
              ? { ...p, connected: true, error: null, usage: data.usage }
              : p
          )
        );
      } else {
        showFeedback(
          data.error || `${providerName} connection failed`,
          "error"
        );
        setProviders((prev) =>
          prev.map((p) =>
            p.provider === providerName
              ? { ...p, connected: false, error: data.error }
              : p
          )
        );
      }
    } catch {
      showFeedback(`Failed to check ${providerName}`, "error");
    } finally {
      setChecking(null);
    }
  };

  const handleConfigure = async (providerName: string, token: string) => {
    try {
      const res = await fetch(goApiUrl("/api/auth/configure"), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ provider: providerName, token }),
      });
      const data = await res.json();
      if (data.ok) {
        showFeedback(`${providerName} configured successfully`, "success");
        setConfiguring(null);
        // Refresh providers
        fetchProviders();
      } else {
        showFeedback(data.error || "Failed to save credentials", "error");
      }
    } catch {
      showFeedback("Failed to save credentials", "error");
    }
  };

  const connectedCount = providers.filter((p) => p.connected).length;
  const totalCount = providers.length;

  return (
    <div className="flex flex-col min-h-full">
      {/* Page Header */}
      <div className="px-6 pt-6 pb-2">
        <h1 className="text-2xl font-heading tracking-wide text-foreground/90">
          Settings
        </h1>
        <p className="text-xs font-mono text-muted-foreground/30 mt-1">
          Platform configuration
        </p>
      </div>

      {/* AI Providers Section */}
      <div className="px-6 pt-4 flex items-start justify-between flex-wrap gap-3">
        <div>
          <div className="flex items-center gap-3">
            <h2 className="text-lg font-heading tracking-wide text-foreground/80">
              AI Providers
            </h2>
            {!loading && (
              <span
                className={`text-[9px] font-mono px-2 py-0.5 rounded-full border ${
                  connectedCount === totalCount && totalCount > 0
                    ? "bg-green-500/15 text-green-400/80 border-green-500/20"
                    : connectedCount > 0
                      ? "bg-amber-500/15 text-amber-400/80 border-amber-500/20"
                      : "bg-white/[0.04] text-muted-foreground/30 border-white/[0.06]"
                }`}
              >
                {connectedCount}/{totalCount} connected
              </span>
            )}
          </div>
          <p className="text-xs font-mono text-muted-foreground/30 mt-1">
            Manage AI provider credentials and monitor usage
          </p>
        </div>

        <button
          onClick={() => fetchProviders()}
          disabled={loading}
          className="flex items-center gap-2 px-4 py-2 rounded-xl text-sm bg-white/[0.04] text-foreground/60 hover:text-foreground/80 hover:bg-white/[0.08] border border-white/[0.06] transition-all disabled:opacity-40"
        >
          <IconRefresh size={16} className={loading ? "animate-spin" : ""} />
          Refresh
        </button>
      </div>

      {/* Feedback toast */}
      {feedback && (
        <div className="px-6 mt-4">
          <div
            className={`rounded-lg px-4 py-2.5 flex items-center gap-2 text-xs font-mono ${
              feedback.type === "success"
                ? "bg-green-500/10 border border-green-500/20 text-green-400"
                : "bg-red-500/10 border border-red-500/20 text-red-400"
            }`}
          >
            {feedback.type === "success" ? (
              <IconCheck size={14} />
            ) : (
              <IconAlertTriangle size={14} />
            )}
            {feedback.message}
          </div>
        </div>
      )}

      {/* Content */}
      <main className="flex-1 px-6 py-6">
        {error && !loading && (
          <div className="rounded-xl bg-red-500/10 border border-red-500/20 px-5 py-4 mb-6 flex items-start gap-3">
            <IconAlertTriangle
              size={18}
              className="text-red-400/70 mt-0.5 shrink-0"
            />
            <div>
              <p className="text-sm text-red-400/80 font-medium">
                Failed to load providers
              </p>
              <p className="text-[11px] text-red-400/50 font-mono mt-1">
                {error}
              </p>
            </div>
          </div>
        )}

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {loading
            ? [1, 2, 3].map((i) => <ProviderCardSkeleton key={i} />)
            : providers.map((provider) => (
                <ProviderCard
                  key={provider.provider}
                  provider={provider}
                  onCheckConnection={() => handleCheck(provider.provider)}
                  onConfigure={() => setConfiguring(provider.provider)}
                  checking={checking === provider.provider}
                />
              ))}
        </div>

        {/* Empty state */}
        {!loading && providers.length === 0 && !error && (
          <div className="flex flex-col items-center justify-center py-16 text-center">
            <div className="w-16 h-16 rounded-2xl bg-white/[0.03] border border-white/[0.06] flex items-center justify-center mb-4">
              <IconPlugConnected
                size={28}
                className="text-muted-foreground/15"
              />
            </div>
            <p className="text-muted-foreground/30 text-lg font-heading">
              No providers detected
            </p>
            <p className="text-muted-foreground/20 text-sm mt-2 font-mono">
              Set up API keys to connect to AI providers
            </p>
          </div>
        )}

        {/* Help text */}
        {!loading && providers.length > 0 && (
          <div className="mt-8 glass-subtle rounded-xl p-5">
            <div className="flex items-center gap-2.5 mb-3">
              <IconKey size={16} className="text-muted-foreground/30" />
              <h3 className="text-xs font-heading tracking-wide text-muted-foreground/50">
                Configuration Help
              </h3>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4 text-[11px] text-muted-foreground/35">
              <div>
                <p className="font-medium text-foreground/50 mb-1">
                  Environment Variables
                </p>
                <p className="font-mono leading-relaxed">
                  Set ANTHROPIC_API_KEY, OPENAI_API_KEY, or GOOGLE_API_KEY in
                  your shell profile.
                </p>
              </div>
              <div>
                <p className="font-medium text-foreground/50 mb-1">CLI</p>
                <p className="font-mono leading-relaxed">
                  Run{" "}
                  <span className="text-blue-400/60">spwn auth</span> to
                  see current status, or use the Configure button above.
                </p>
              </div>
              <div>
                <p className="font-medium text-foreground/50 mb-1">
                  Keychain
                </p>
                <p className="font-mono leading-relaxed">
                  spwn can store credentials in your system keychain for
                  enhanced security.
                </p>
              </div>
            </div>
          </div>
        )}
      </main>

      {/* Configure Modal */}
      {configuring && (
        <ConfigureModal
          provider={configuring}
          onClose={() => setConfiguring(null)}
          onSave={(token) => handleConfigure(configuring, token)}
        />
      )}
    </div>
  );
}
