"use client";

import { useState, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import {
  CommandDialog,
  Command,
  CommandInput,
  CommandList,
  CommandEmpty,
  CommandGroup,
  CommandItem,
  CommandSeparator,
} from "@/components/ui/command";
import {
  IconWorldFilled,
  IconUserFilled,
  IconRocket,
  IconBrain,
  IconCamera,
  IconTrash,
  IconPlus,
  IconBook2,
  IconPackage,
} from "@tabler/icons-react";
import type { World } from "@/lib/types";
import { apiGet } from "@/lib/api-client";

interface AgentListItem {
  name: string;
  path: string;
  layers: Record<string, string[]>;
}

export function CommandPalette() {
  const [open, setOpen] = useState(false);
  const [worlds, setWorlds] = useState<World[]>([]);
  const [agents, setAgents] = useState<AgentListItem[]>([]);
  const router = useRouter();

  // Listen for Cmd+K
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        e.preventDefault();
        setOpen((prev) => !prev);
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, []);

  // Fetch data when opening
  useEffect(() => {
    if (!open) return;
    Promise.all([
      apiGet<World[]>("/api/universes").catch(() => [] as World[]),
      apiGet<AgentListItem[]>("/api/agents").catch(() => [] as AgentListItem[]),
    ]).then(([w, a]) => {
      setWorlds(w ?? []);
      setAgents(a ?? []);
    });
  }, [open]);

  const navigate = useCallback(
    (path: string) => {
      setOpen(false);
      router.push(path);
    },
    [router]
  );

  function extractName(id: string): string {
    const parts = id.split("-");
    return parts.length >= 2
      ? parts[1].charAt(0).toUpperCase() + parts[1].slice(1)
      : id;
  }

  return (
    <CommandDialog open={open} onOpenChange={setOpen}>
      <Command className="rounded-xl border border-white/[0.08] bg-popover/95 backdrop-blur-md">
        <CommandInput placeholder="Search worlds, agents, commands..." />
        <CommandList>
          <CommandEmpty>No results found.</CommandEmpty>

          {/* Worlds */}
          {worlds.length > 0 && (
            <CommandGroup heading="Worlds">
              {worlds.map((world) => (
                <CommandItem
                  key={world.id}
                  onSelect={() => navigate(`/world/${world.id}`)}
                >
                  <IconWorldFilled size={14} className="text-muted-foreground/50" />
                  <span>{extractName(world.id)}</span>
                  <span className="flex-1 text-right -mr-6 text-[10px] font-mono text-muted-foreground/30">
                    {world.agents.length} agents · {world.status}
                  </span>
                </CommandItem>
              ))}
            </CommandGroup>
          )}

          {/* Agents */}
          {agents.length > 0 && (
            <>
              <CommandSeparator />
              <CommandGroup heading="Agents">
                {agents.map((agent) => {
                  // Find which world this agent is in
                  const agentWorld = worlds.find((w) =>
                    w.agents.some((a) => a.name === agent.name)
                  );
                  const href = agentWorld
                    ? `/agents/${encodeURIComponent(agent.name)}?world=${agentWorld.id}`
                    : `/agents/${encodeURIComponent(agent.name)}`;
                  return (
                    <CommandItem
                      key={agent.name}
                      onSelect={() => navigate(href)}
                    >
                      <IconUserFilled size={14} className="text-muted-foreground/50" />
                      <span>{agent.name}</span>
                      {agentWorld && (
                        <span className="flex-1 text-right -mr-6 text-[10px] font-mono text-muted-foreground/30">
                          in {extractName(agentWorld.id)}
                        </span>
                      )}
                      {!agentWorld && (
                        <span className="flex-1 text-right -mr-6 text-[10px] font-mono text-muted-foreground/20">
                          limbo
                        </span>
                      )}
                    </CommandItem>
                  );
                })}
              </CommandGroup>
            </>
          )}

          {/* Navigation */}
          <CommandSeparator />
          <CommandGroup heading="Navigation">
            <CommandItem onSelect={() => navigate("/")}>
              <IconWorldFilled size={14} className="text-muted-foreground/50" />
              <span>Go to Dashboard</span>
            </CommandItem>
            <CommandItem onSelect={() => navigate("/architect")}>
              <IconBrain size={14} className="text-muted-foreground/50" />
              <span>Go to Architect</span>
            </CommandItem>
            <CommandItem onSelect={() => navigate("/knowledge")}>
              <IconBook2 size={14} className="text-muted-foreground/50" />
              <span>Go to Knowledge</span>
            </CommandItem>
            {/* Marketplace — hidden until ready */}
          </CommandGroup>

          {/* Actions */}
          <CommandSeparator />
          <CommandGroup heading="Actions">
            <CommandItem onSelect={() => {
              setOpen(false);
              // Trigger spawn world dialog on home page
              navigate("/");
              setTimeout(() => {
                window.dispatchEvent(new KeyboardEvent("keydown", { key: "n", metaKey: true, bubbles: true }));
              }, 100);
            }}>
              <IconRocket size={14} className="text-muted-foreground/50" />
              <span>Spawn World</span>
              <span className="flex-1 text-right -mr-6 text-[10px] font-mono text-muted-foreground/20">⌘N</span>
            </CommandItem>
            <CommandItem onSelect={() => {
              setOpen(false);
              const name = prompt("Agent name:");
              if (name?.trim()) {
                navigate(`/agents/${name.trim()}`);
              }
            }}>
              <IconPlus size={14} className="text-muted-foreground/50" />
              <span>Create Agent</span>
            </CommandItem>
          </CommandGroup>
        </CommandList>
      </Command>
    </CommandDialog>
  );
}
