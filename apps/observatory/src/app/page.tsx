"use client";

import { useEffect, useState } from "react";
import { Planet } from "@/components/planet";
import { Header } from "@/components/header";

interface Agent {
  name: string;
  tier: string;
  status: string;
}

export interface World {
  id: string;
  config: string;
  agent: string;
  agents: Agent[];
  status: "running" | "idle" | "stopped" | "creating";
  created_at: string;
  workspace: string;
}

const MOCK_WORLDS: World[] = [
  {
    id: "w-titan-84721",
    config: "default",
    agent: "neo",
    agents: [{ name: "neo", tier: "citizen", status: "running" }],
    status: "running",
    created_at: new Date(Date.now() - 1000 * 60 * 12).toISOString(),
    workspace: "~/acme-api",
  },
  {
    id: "w-europa-39205",
    config: "default",
    agent: "morpheus",
    agents: [
      { name: "morpheus", tier: "governor", status: "running" },
      { name: "trinity", tier: "citizen", status: "idle" },
    ],
    status: "running",
    created_at: new Date(Date.now() - 1000 * 60 * 45).toISOString(),
    workspace: "~/signews",
  },
  {
    id: "w-ganymede-51003",
    config: "backend",
    agent: "atlas",
    agents: [{ name: "atlas", tier: "citizen", status: "idle" }],
    status: "idle",
    created_at: new Date(Date.now() - 1000 * 60 * 120).toISOString(),
    workspace: "~/infra",
  },
];

export default function UniversePage() {
  const [worlds, setWorlds] = useState<World[]>([]);
  const [selected, setSelected] = useState(0);

  useEffect(() => {
    setWorlds(MOCK_WORLDS);
  }, []);

  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      if (worlds.length === 0) return;
      if (e.key === "ArrowRight" || e.key === "d") {
        setSelected((s) => (s + 1) % worlds.length);
      } else if (e.key === "ArrowLeft" || e.key === "a") {
        setSelected((s) => (s - 1 + worlds.length) % worlds.length);
      } else if (e.key === "Enter") {
        // TODO: navigate into world
      }
    };
    window.addEventListener("keydown", handleKey);
    return () => window.removeEventListener("keydown", handleKey);
  }, [worlds.length]);

  return (
    <div className="flex flex-col h-screen">
      <Header worldCount={worlds.length} />

      <main className="flex-1 relative flex items-center justify-center">
        {worlds.length === 0 ? (
          <div className="text-center">
            <p className="text-[rgba(255,255,255,0.3)] text-lg font-heading">
              No active worlds
            </p>
            <p className="text-[rgba(255,255,255,0.2)] text-sm mt-2 font-mono">
              spwn up --agent neo -w .
            </p>
          </div>
        ) : (
          <div className="flex items-end gap-12 md:gap-20">
            {worlds.map((world, i) => (
              <Planet
                key={world.id}
                world={world}
                index={i}
                isSelected={selected === i}
                onClick={() => setSelected(i)}
              />
            ))}
          </div>
        )}
      </main>
    </div>
  );
}
