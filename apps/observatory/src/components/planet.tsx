"use client";

import createGlobe from "cobe";
import { useEffect, useRef } from "react";
import type { World } from "@/app/page";

interface PlanetProps {
  world: World;
  index: number;
  onClick: () => void;
  isSelected: boolean;
}

const STATUS_CONFIG: Record<
  string,
  { base: [number, number, number]; glow: [number, number, number]; marker: [number, number, number]; brightness: number }
> = {
  running: {
    base: [0.12, 0.12, 0.12],
    glow: [0.08, 0.15, 0.08],
    marker: [0.4, 1, 0.4],
    brightness: 8,
  },
  idle: {
    base: [0.12, 0.12, 0.1],
    glow: [0.12, 0.1, 0.04],
    marker: [1, 0.85, 0.3],
    brightness: 5,
  },
  stopped: {
    base: [0.08, 0.08, 0.08],
    glow: [0.04, 0.04, 0.04],
    marker: [0.5, 0.5, 0.5],
    brightness: 3,
  },
  creating: {
    base: [0.1, 0.1, 0.14],
    glow: [0.06, 0.08, 0.15],
    marker: [0.4, 0.6, 1],
    brightness: 6,
  },
};

const STATUS_DOT_CSS: Record<string, string> = {
  running: "#22c55e",
  idle: "#eab308",
  stopped: "rgba(255,255,255,0.2)",
  creating: "#60a5fa",
};

const TIER_ICON: Record<string, string> = {
  governor: "♛",
  citizen: "◉",
  npc: "◌",
};

function extractName(id: string): string {
  const parts = id.split("-");
  return parts.length >= 2
    ? parts[1].charAt(0).toUpperCase() + parts[1].slice(1)
    : id;
}

function timeAgo(iso: string): string {
  const d = Date.now() - new Date(iso).getTime();
  const m = Math.floor(d / 60000);
  if (m < 60) return `${m}m`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h`;
  return `${Math.floor(h / 24)}d`;
}

// Deterministic seed from string so each planet/agent has unique position
function hashCode(s: string): number {
  let h = 0;
  for (let i = 0; i < s.length; i++) {
    h = (Math.imul(31, h) + s.charCodeAt(i)) | 0;
  }
  return Math.abs(h);
}

// Generate a deterministic globe coordinate from an agent name
function agentToLocation(name: string): [number, number] {
  const h = hashCode(name);
  const lat = ((h % 1000) / 1000) * 140 - 70; // -70 to 70
  const lng = (((h >> 10) % 1000) / 1000) * 360 - 180; // -180 to 180
  return [lat, lng];
}

// Bright marker colors per agent tier
const MARKER_COLORS: Record<string, [number, number, number]> = {
  governor: [1, 0.85, 0.2],   // gold
  citizen: [0.3, 0.9, 0.5],   // green
  npc: [0.5, 0.7, 1],         // blue
};

export function Planet({ world, index, onClick, isSelected }: PlanetProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const globeRef = useRef<any>(null);
  const phiRef = useRef(hashCode(world.id) % 628 / 100); // unique starting angle
  const config = STATUS_CONFIG[world.status] ?? STATUS_CONFIG.stopped;
  const name = extractName(world.id);
  const size = isSelected ? 240 : 180;

  useEffect(() => {
    if (!canvasRef.current) return;

    const globe = createGlobe(canvasRef.current, {
      devicePixelRatio: 2,
      width: size * 2,
      height: size * 2,
      phi: phiRef.current,
      theta: 0.15 + index * 0.3,
      dark: 1,
      diffuse: 1.4,
      mapSamples: 16000,
      mapBrightness: config.brightness,
      mapBaseBrightness: 0.02,
      baseColor: config.base,
      markerColor: [1, 1, 1],
      glowColor: config.glow,
      markers: world.agents.map((a) => ({
        location: agentToLocation(a.name),
        size: a.tier === "governor" ? 0.12 : 0.08,
      })),
      markerElevation: 0,
    });
    globeRef.current = globe;

    // Auto-rotate via requestAnimationFrame
    let raf: number;
    const spin = () => {
      phiRef.current += isSelected ? 0.004 : 0.002;
      globe.update({ phi: phiRef.current });
      raf = requestAnimationFrame(spin);
    };
    raf = requestAnimationFrame(spin);

    return () => {
      cancelAnimationFrame(raf);
      globe.destroy();
    };
  }, [world.id, world.status, isSelected, size]);

  return (
    <button
      onClick={onClick}
      className="group relative flex flex-col items-center transition-all duration-500 focus:outline-none"
      style={{
        transform: isSelected ? "translateY(-20px)" : "translateY(0)",
      }}
    >
      {/* ── Top HUD: name + status ── */}
      <div
        className="mb-3 text-center transition-all duration-300"
        style={{ opacity: isSelected ? 1 : 0.5 }}
      >
        <p
          className="font-heading tracking-wide transition-all duration-300"
          style={{
            color: isSelected
              ? "rgba(255,255,255,0.95)"
              : "rgba(255,255,255,0.45)",
            fontSize: isSelected ? "1.1rem" : "0.85rem",
          }}
        >
          {name}
        </p>
        <div className="flex items-center justify-center gap-1.5 mt-1">
          <div
            className="w-1.5 h-1.5 rounded-full"
            style={{
              backgroundColor: STATUS_DOT_CSS[world.status],
              boxShadow: isSelected
                ? `0 0 6px ${STATUS_DOT_CSS[world.status]}`
                : "none",
            }}
          />
          <span className="font-mono text-[10px] uppercase tracking-widest text-[rgba(255,255,255,0.3)]">
            {world.status}
          </span>
        </div>
      </div>

      {/* ── Globe ── */}
      <div
        className="relative transition-all duration-500"
        style={{
          width: size,
          height: size,
          filter: isSelected
            ? `drop-shadow(0 0 24px ${STATUS_DOT_CSS[world.status]}40)`
            : "none",
        }}
      >
        <canvas
          ref={canvasRef}
          style={{
            width: size,
            height: size,
            opacity: isSelected ? 1 : 0.7,
            transition: "opacity 0.5s",
          }}
        />
      </div>

      {/* ── Bottom HUD: metadata (selected only) ── */}
      <div
        className="mt-3 transition-all duration-300 w-48"
        style={{
          opacity: isSelected ? 1 : 0,
          transform: isSelected ? "translateY(0)" : "translateY(8px)",
          pointerEvents: isSelected ? "auto" : "none",
        }}
      >
        <div className="glass-subtle px-3 py-2.5 space-y-1.5">
          <div className="flex justify-between items-center">
            <span className="text-[10px] uppercase tracking-wider text-[rgba(255,255,255,0.25)]">
              Agents
            </span>
            <div className="flex items-center gap-1.5">
              {world.agents.map((a) => (
                <span
                  key={a.name}
                  className="text-[10px] font-mono text-[rgba(255,255,255,0.55)]"
                >
                  {TIER_ICON[a.tier] ?? "◌"}{" "}{a.name}
                </span>
              ))}
            </div>
          </div>
          <div className="flex justify-between items-center">
            <span className="text-[10px] uppercase tracking-wider text-[rgba(255,255,255,0.25)]">
              Config
            </span>
            <span className="text-[10px] font-mono text-[rgba(255,255,255,0.45)]">
              {world.config}
            </span>
          </div>
          <div className="flex justify-between items-center">
            <span className="text-[10px] uppercase tracking-wider text-[rgba(255,255,255,0.25)]">
              Path
            </span>
            <span className="text-[10px] font-mono text-[rgba(255,255,255,0.45)] truncate max-w-28">
              {world.workspace}
            </span>
          </div>
          <div className="flex justify-between items-center">
            <span className="text-[10px] uppercase tracking-wider text-[rgba(255,255,255,0.25)]">
              Uptime
            </span>
            <span className="text-[10px] font-mono text-[rgba(255,255,255,0.45)]">
              {timeAgo(world.created_at)}
            </span>
          </div>
        </div>

        <button
          className="mt-2 w-full py-1.5 text-[11px] font-mono uppercase tracking-widest text-[rgba(255,255,255,0.45)] border border-[rgba(255,255,255,0.08)] rounded-md transition-all duration-200 hover:text-white hover:border-[rgba(255,255,255,0.25)] hover:bg-[rgba(255,255,255,0.04)] active:scale-[0.97]"
          onClick={(e) => {
            e.stopPropagation();
          }}
        >
          ↵ Enter World
        </button>
      </div>
    </button>
  );
}
