"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { Planet } from "@/components/planet";
import { MOCK_WORLDS } from "@/lib/mock-data";

export interface World {
  id: string;
  config: string;
  agent: string;
  agents: { name: string; tier: string; status: string }[];
  status: "running" | "idle" | "stopped" | "creating";
  created_at: string;
  workspace: string;
}

export default function UniverseMapPage() {
  const [worlds, setWorlds] = useState<World[]>([]);
  const [selected, setSelected] = useState(0);
  const router = useRouter();

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
        router.push(`/world/${worlds[selected].id}`);
      }
    };
    window.addEventListener("keydown", handleKey);
    return () => window.removeEventListener("keydown", handleKey);
  }, [worlds, selected, router]);

  return (
    <div className="flex flex-col min-h-screen">
      {/* Minimal header */}
      <header className="relative z-20 flex items-center justify-between px-6 py-4">
        <div className="flex items-center gap-3">
          <span className="text-lg tracking-[0.2em] font-heading text-foreground/90">
            ⬡ observatory
          </span>
          <span className="text-xs font-mono text-muted-foreground/30">spwn</span>
        </div>
        <div className="glass-subtle px-3 py-1.5 flex items-center gap-2">
          <div className="w-1.5 h-1.5 rounded-full bg-green-500 animate-pulse" />
          <span className="text-xs font-mono text-muted-foreground">
            {worlds.length} world{worlds.length !== 1 ? "s" : ""}
          </span>
        </div>
      </header>

      {/* Universe canvas */}
      <main className="flex-1 flex items-center justify-center py-16">
        {worlds.length === 0 ? (
          <div className="text-center">
            <p className="text-muted-foreground/30 text-lg font-heading">No active worlds</p>
            <p className="text-muted-foreground/20 text-sm mt-2 font-mono">spwn up --agent neo -w .</p>
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
    </div>
  );
}
