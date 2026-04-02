"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { Planet } from "@/components/planet";
import { AVAILABLE_CONFIGS } from "@/lib/types";
import type { World } from "@/lib/types";
import { IconPlus, IconRocket, IconX } from "@tabler/icons-react";
import { Skeleton } from "@/components/ui/skeleton";
import { apiGet } from "@/lib/api-client";

export default function UniverseMapPage() {
  const [worlds, setWorlds] = useState<World[]>([]);
  const [selected, setSelected] = useState(0);
  const [showSpawn, setShowSpawn] = useState(false);
  const [loading, setLoading] = useState(true);
  const router = useRouter();

  const fetchWorlds = () => {
    apiGet<World[]>("/api/universes", "/api/worlds")
      .then((data) => {
        setWorlds(data ?? []);
        setLoading(false);
      })
      .catch(() => {
        setWorlds([]);
        setLoading(false);
      });
  };

  useEffect(() => {
    fetchWorlds();
    const interval = setInterval(fetchWorlds, 5000);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      if (showSpawn) return;
      if (worlds.length === 0) return;
      if (e.key === "ArrowRight" || e.key === "d") {
        setSelected((s) => (s + 1) % worlds.length);
      } else if (e.key === "ArrowLeft" || e.key === "a") {
        setSelected((s) => (s - 1 + worlds.length) % worlds.length);
      } else if (e.key === "Enter") {
        router.push(`/world/${worlds[selected].id}`);
      }
    };
    window.addEventListener("keydown", handleKey);
    return () => window.removeEventListener("keydown", handleKey);
  }, [worlds, selected, router, showSpawn]);

  return (
    <div className="flex flex-col min-h-screen">
      {/* Universe header */}
      <div className="px-8 pt-8 flex items-start justify-between">
        <div>
          <h1 className="text-3xl font-heading tracking-wide text-foreground/90">Meson</h1>
          <p className="text-[11px] font-mono text-muted-foreground/30 mt-1.5 flex items-center gap-3">
            <span>orbstack</span>
            <span className="text-muted-foreground/15">·</span>
            <span>docker v28.5.2</span>
            <span className="text-muted-foreground/15">·</span>
            <span>claude-code</span>
            <span className="text-muted-foreground/15">·</span>
            <span>{worlds.length} worlds</span>
            <span className="text-muted-foreground/15">·</span>
            <span>{worlds.reduce((n, w) => n + w.agents.length, 0)} agents</span>
            <span className="text-muted-foreground/15">·</span>
            <span>~/.spwn</span>
          </p>
        </div>

        {/* Spawn World button */}
        <button
          onClick={() => setShowSpawn(true)}
          className="flex items-center gap-2 px-4 py-2 rounded-xl text-sm bg-white/[0.04] text-foreground/60 hover:text-foreground/80 hover:bg-white/[0.08] border border-white/[0.06] transition-all"
        >
          <IconPlus size={16} />
          Spawn World
        </button>
      </div>

      <main className="flex-1 flex items-center justify-center py-16">
        {loading ? (
          <div className="flex items-center gap-12 md:gap-20">
            {[1, 2, 3].map((i) => (
              <div key={i} className="flex flex-col items-center gap-4">
                <Skeleton className="w-24 h-24 rounded-full" />
                <Skeleton className="h-4 w-16" />
                <Skeleton className="h-3 w-24" />
              </div>
            ))}
          </div>
        ) : worlds.length === 0 ? (
          <div className="text-center">
          <p className="text-muted-foreground/30 text-lg font-heading">No worlds running</p>
          <p className="text-muted-foreground/20 text-sm mt-2 font-mono">Spawn one to get started</p>
            <button
              onClick={() => setShowSpawn(true)}
              className="mt-6 flex items-center gap-2 px-5 py-2.5 rounded-xl text-sm mx-auto bg-white/[0.04] text-foreground/60 hover:text-foreground/80 hover:bg-white/[0.08] border border-white/[0.06] transition-all"
            >
              <IconRocket size={16} />
              Spawn your first world
            </button>
          </div>
        ) : (
          <div className="flex items-center gap-12 md:gap-20">
            {worlds.map((world, i) => (
              <Planet
                key={world.id}
                world={world}
                index={i}
                isSelected={selected === i}
                onClick={() => setSelected(i)}
                onEnter={() => router.push(`/world/${worlds[i].id}`)}
              />
            ))}
          </div>
        )}
      </main>

      {/* Spawn World Dialog */}
      {showSpawn && (
        <SpawnWorldDialog onClose={() => setShowSpawn(false)} />
      )}
    </div>
  );
}

interface AgentListItem {
  name: string;
  path: string;
  layers: Record<string, string[]>;
}

function SpawnWorldDialog({ onClose }: { onClose: () => void }) {
  const router = useRouter();
  const [agentName, setAgentName] = useState("");
  const [workspace, setWorkspace] = useState("~/");
  const [config, setConfig] = useState("default");
  const [tier, setTier] = useState("citizen");
  const [spawning, setSpawning] = useState(false);
  const [availableAgents, setAvailableAgents] = useState<AgentListItem[]>([]);

  const [error, setError] = useState("");

  // Fetch available agents for dropdown
  useEffect(() => {
    apiGet<AgentListItem[]>("/api/agents", "/api/agents")
      .then((agents) => {
        setAvailableAgents(agents ?? []);
        if (agents && agents.length > 0 && !agentName) {
          setAgentName(agents[0].name);
        }
      })
      .catch(() => {});
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleSpawn = async () => {
    setSpawning(true);
    setError("");
    try {
      const res = await fetch("http://localhost:3001/api/worlds", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ agent: agentName.trim(), workspace, config, tier }),
        signal: AbortSignal.timeout(30000),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        setError(data.error || "Failed to spawn world");
        setSpawning(false);
        return;
      }
      onClose();
      // Redirect to the new world if we got an ID back
      if (data.id) {
        router.push(`/world/${data.id}`);
      }
    } catch {
      setError("Failed to connect to API");
      setSpawning(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={onClose} />

      {/* Dialog */}
      <div className="relative z-10 w-full max-w-md mx-4 rounded-2xl bg-popover/95 backdrop-blur-md border border-white/[0.08] shadow-2xl">
        {/* Header */}
        <div className="px-6 pt-6 pb-4 flex items-center justify-between">
          <div>
            <h2 className="text-lg font-heading text-foreground/90">Spawn World</h2>
            <p className="text-[11px] text-muted-foreground/40 mt-0.5">Create a new isolated world for your agent</p>
          </div>
          <button
            onClick={onClose}
            className="text-muted-foreground/40 hover:text-foreground/60 transition-colors"
          >
            <IconX size={18} />
          </button>
        </div>

        {/* Form */}
        <div className="px-6 pb-6 space-y-4">
          {/* Agent name — dropdown if agents exist, text input as fallback */}
          <div>
            <label className="text-[10px] uppercase tracking-widest text-muted-foreground/40 block mb-1.5">
              Agent
            </label>
            {availableAgents.length > 0 ? (
              <select
                value={agentName}
                onChange={(e) => setAgentName(e.target.value)}
                className="w-full bg-white/[0.03] border border-white/[0.08] rounded-lg px-3 py-2.5 text-sm text-foreground/80 focus:outline-none focus:border-white/[0.15] transition-colors"
                autoFocus
              >
                {availableAgents.map((a) => (
                  <option key={a.name} value={a.name}>{a.name}</option>
                ))}
              </select>
            ) : (
              <input
                value={agentName}
                onChange={(e) => setAgentName(e.target.value)}
                placeholder="e.g. neo, morpheus, atlas..."
                className="w-full bg-white/[0.03] border border-white/[0.08] rounded-lg px-3 py-2.5 text-sm text-foreground/80 placeholder:text-muted-foreground/25 focus:outline-none focus:border-white/[0.15] transition-colors"
                autoFocus
              />
            )}
          </div>

          {/* Workspace */}
          <div>
            <label className="text-[10px] uppercase tracking-widest text-muted-foreground/40 block mb-1.5">
              Workspace Path
            </label>
            <input
              value={workspace}
              onChange={(e) => setWorkspace(e.target.value)}
              placeholder="~/my-project"
              className="w-full bg-white/[0.03] border border-white/[0.08] rounded-lg px-3 py-2.5 text-sm font-mono text-foreground/80 placeholder:text-muted-foreground/25 focus:outline-none focus:border-white/[0.15] transition-colors"
            />
          </div>

          {/* Config + Tier row */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="text-[10px] uppercase tracking-widest text-muted-foreground/40 block mb-1.5">
                Config
              </label>
              <select
                value={config}
                onChange={(e) => setConfig(e.target.value)}
                className="w-full bg-white/[0.03] border border-white/[0.08] rounded-lg px-3 py-2.5 text-sm text-foreground/80 focus:outline-none focus:border-white/[0.15] transition-colors"
              >
                {AVAILABLE_CONFIGS.map((c) => (
                  <option key={c} value={c}>{c}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="text-[10px] uppercase tracking-widest text-muted-foreground/40 block mb-1.5">
                Agent Tier
              </label>
              <select
                value={tier}
                onChange={(e) => setTier(e.target.value)}
                className="w-full bg-white/[0.03] border border-white/[0.08] rounded-lg px-3 py-2.5 text-sm text-foreground/80 focus:outline-none focus:border-white/[0.15] transition-colors"
              >
                <option value="governor">Governor</option>
                <option value="citizen">Citizen</option>
                <option value="npc">NPC</option>
              </select>
            </div>
          </div>

          {/* CLI preview */}
          <div className="rounded-lg bg-white/[0.02] border border-white/[0.05] px-3 py-2.5 font-mono text-[11px] text-muted-foreground/35">
            spwn up --agent {agentName || "‹name›"} --tier {tier} --config {config} -w {workspace}
          </div>

          {/* Error display */}
          {error && (
            <div className="rounded-lg bg-red-500/10 border border-red-500/20 px-3 py-2 text-xs text-red-400 font-mono">
              {error}
            </div>
          )}

          {/* Spawn button */}
          <button
            onClick={handleSpawn}
            disabled={!agentName.trim() || spawning}
            className="w-full flex items-center justify-center gap-2 py-3 rounded-xl text-sm font-medium bg-white/[0.06] text-foreground/70 hover:bg-white/[0.1] hover:text-foreground/90 border border-white/[0.08] transition-all disabled:opacity-30 disabled:cursor-not-allowed"
          >
            {spawning ? (
              <>
                <div className="w-3.5 h-3.5 border-2 border-foreground/30 border-t-foreground/70 rounded-full animate-spin" />
                Spawning...
              </>
            ) : (
              <>
                <IconRocket size={16} />
                Spawn World
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  );
}
