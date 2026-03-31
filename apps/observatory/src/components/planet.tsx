"use client";

import createGlobe from "cobe";
import { useEffect, useRef } from "react";
import type { World } from "@/lib/mock-data";

interface PlanetProps {
  world: World;
  index: number;
  onClick: () => void;
  onEnter?: () => void;
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


export function Planet({ world, index, onClick, onEnter, isSelected }: PlanetProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const globeRef = useRef<any>(null);
  const phiRef = useRef(hashCode(world.id) % 628 / 100); // unique starting angle
  const config = STATUS_CONFIG[world.status] ?? STATUS_CONFIG.stopped;
  const name = extractName(world.id);
  const size = 200; // constant canvas size — CSS scale handles selection

  const wrapperRef = useRef<HTMLDivElement>(null);
  const glowRefs = useRef<Map<string, HTMLDivElement>>(new Map());
  const selectedRef = useRef(isSelected);
  selectedRef.current = isSelected;

  useEffect(() => {
    if (!canvasRef.current) return;

    const globe = createGlobe(canvasRef.current, {
      devicePixelRatio: 2,
      width: size * 2,
      height: size * 2,
      phi: phiRef.current,
      theta: 0.15 + index * 0.3,
      dark: 1,
      diffuse: 1.2 + (hashCode(world.id) % 10) * 0.08,
      mapSamples: 16000,
      mapBrightness: 6,
      mapBaseBrightness: 0.02,
      baseColor: config.base,
      markerColor: [1, 1, 1],
      glowColor: config.glow,
      markers: world.agents.map((a) => ({
        location: agentToLocation(a.name),
        size: 0.01,
        id: a.name,
      })),
      markerElevation: 0,
    });
    globeRef.current = globe;

    // Auto-rotate with gentle oscillation when selected
    let raf: number;
    let frame = 0;
    const spin = () => {
      frame++;
      const sel = selectedRef.current;
      phiRef.current += sel ? 0.005 : 0.002;

      // Selected: gentle theta wobble + breathing diffuse
      const theta = sel
        ? 0.15 + index * 0.3 + Math.sin(frame * 0.008) * 0.12
        : 0.15 + index * 0.3;
      const diffuse = sel
        ? 1.4 + Math.sin(frame * 0.015) * 0.4
        : 1.2 + (hashCode(world.id) % 10) * 0.08;
      const scale = sel
        ? 1.0 + Math.sin(frame * 0.012) * 0.015
        : 1.0;

      globe.update({ phi: phiRef.current, theta, diffuse, scale });

      // Sync glow overlay positions from cobe's auto-generated anchor divs
      const wrapper = wrapperRef.current;
      if (wrapper) {
        world.agents.forEach((a) => {
          const anchor = wrapper.querySelector(
            `[style*="--cobe-marker-${a.name}"]`
          ) as HTMLElement | null;
          const glow = glowRefs.current.get(a.name);
          if (anchor && glow) {
            glow.style.left = anchor.style.left;
            glow.style.top = anchor.style.top;
            // Check visibility via computed custom property
            const vis = getComputedStyle(anchor).getPropertyValue(
              `--cobe-marker-visible-${a.name}`
            );
            glow.style.opacity = vis ? "1" : "0";
          }
        });
      }

      raf = requestAnimationFrame(spin);
    };
    raf = requestAnimationFrame(spin);

    return () => {
      cancelAnimationFrame(raf);
      globe.destroy();
    };
  }, [world.id, world.status]);

  return (
    <div
      onClick={onClick}
      role="button"
      tabIndex={0}
      className="group relative flex flex-col items-center transition-all duration-500 focus:outline-none cursor-pointer"
    >
      {/* ── Top HUD: name + status ── */}
      <div
        className="text-center transition-all duration-500"
        style={{ marginBottom: isSelected ? 64 : 12, opacity: isSelected ? 1 : 0.4 }}
      >
        <p
          className="font-heading tracking-wider transition-all duration-500"
          style={{
            color: isSelected
              ? "rgba(255,255,255,0.95)"
              : "rgba(255,255,255,0.4)",
            fontSize: isSelected ? "1.15rem" : "0.8rem",
            letterSpacing: isSelected ? "0.12em" : "0.05em",
          }}
        >
          {name}
        </p>
        <div className="flex items-center justify-center gap-2 mt-1.5">
          <div
            className="w-1.5 h-1.5 rounded-full transition-all duration-500"
            style={{
              backgroundColor: STATUS_DOT_CSS[world.status],
              boxShadow: isSelected
                ? `0 0 8px ${STATUS_DOT_CSS[world.status]}`
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
        ref={wrapperRef}
        className="relative transition-all duration-700 ease-out"
        style={{
          width: size,
          height: size,
          transform: `scale(${isSelected ? 1.8 : 0.85})`,
          filter: isSelected
            ? `brightness(1.1) drop-shadow(0 0 24px ${STATUS_DOT_CSS[world.status]}40)`
            : "blur(1.5px) brightness(0.6)",
        }}
      >
        <canvas
          ref={canvasRef}
          style={{
            width: size,
            height: size,
            transition: "opacity 0.5s",
          }}
        />
        {/* Glow overlays — positioned by JS sync from cobe anchors */}
        {world.agents.map((a) => (
          <div
            key={`glow-${a.name}`}
            ref={(el) => {
              if (el) glowRefs.current.set(a.name, el);
            }}
            className="absolute pointer-events-none"
            style={{
              width: 28,
              height: 28,
              transform: "translate(-50%, -50%)",
              opacity: 0,
              background: "radial-gradient(circle, rgba(255,255,255,0.95) 0%, rgba(255,255,255,0.35) 20%, rgba(255,255,255,0) 65%)",
              boxShadow: "0 0 12px 5px rgba(255,255,255,0.5), 0 0 28px 10px rgba(255,255,255,0.15)",
              borderRadius: "50%",
              transition: "opacity 0.3s",
            }}
          />
        ))}
      </div>

      {/* ── Bottom HUD: metadata (selected only) ── */}
      <div
        className="transition-all duration-500 w-52"
        style={{
          marginTop: isSelected ? 64 : 12,
          opacity: isSelected ? 1 : 0,
          transform: isSelected ? "translateY(0)" : "translateY(12px)",
          pointerEvents: isSelected ? "auto" : "none",
        }}
      >
        {/* Agents row */}
        <div className="flex items-center justify-center gap-3 mb-3">
          {world.agents.map((a) => (
            <div key={a.name} className="flex items-center gap-1.5">
              <div
                className="w-1 h-1 rounded-full"
                style={{
                  backgroundColor: STATUS_DOT_CSS[a.status] ?? STATUS_DOT_CSS.stopped,
                  boxShadow: `0 0 4px ${STATUS_DOT_CSS[a.status] ?? STATUS_DOT_CSS.stopped}`,
                }}
              />
              <span className="text-[11px] font-mono text-[rgba(255,255,255,0.6)]">
                {a.name}
              </span>
              <span className="text-[9px] text-[rgba(255,255,255,0.25)]">
                {TIER_ICON[a.tier] ?? "◌"}
              </span>
            </div>
          ))}
        </div>

        {/* Stat grid */}
        <div className="glass-subtle px-4 py-3">
          <div className="grid grid-cols-3 gap-y-2.5 text-center">
            <div>
              <p className="text-[9px] uppercase tracking-widest text-[rgba(255,255,255,0.2)] mb-0.5">Config</p>
              <p className="text-[11px] font-mono text-[rgba(255,255,255,0.6)]">{world.config}</p>
            </div>
            <div>
              <p className="text-[9px] uppercase tracking-widest text-[rgba(255,255,255,0.2)] mb-0.5">Uptime</p>
              <p className="text-[11px] font-mono text-[rgba(255,255,255,0.6)]">{timeAgo(world.created_at)}</p>
            </div>
            <div>
              <p className="text-[9px] uppercase tracking-widest text-[rgba(255,255,255,0.2)] mb-0.5">Agents</p>
              <p className="text-[11px] font-mono text-[rgba(255,255,255,0.6)]">{world.agents.length}</p>
            </div>
          </div>
        </div>

        {/* Path */}
        <p className="text-center text-[10px] font-mono text-[rgba(255,255,255,0.2)] mt-2 truncate px-2">
          {world.workspace}
        </p>

        {/* Enter button */}
        <button
          className="mt-3 w-full py-2 text-[11px] font-mono uppercase tracking-[0.2em] text-[rgba(255,255,255,0.4)] glass-subtle transition-all duration-200 hover:text-white hover:border-[rgba(255,255,255,0.2)] active:scale-[0.97]"
          onClick={(e) => {
            e.stopPropagation();
            onEnter?.();
          }}
        >
          ↵ Enter
        </button>
      </div>
    </div>
  );
}
