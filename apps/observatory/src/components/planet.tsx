"use client";

import createGlobe from "cobe";
import { useEffect, useRef, useState } from "react";
import type { World } from "@/lib/types";

interface PlanetProps {
  world: World;
  index: number;
  onClick: () => void;
  onEnter?: () => void;
  isSelected: boolean;
  compact?: boolean;
  hideLabels?: boolean;
}

const STATUS_CONFIG: Record<
  string,
  { base: [number, number, number]; glow: [number, number, number]; marker: [number, number, number]; brightness: number }
> = {
  running: {
    base: [0.18, 0.2, 0.18],
    glow: [0.12, 0.25, 0.12],
    marker: [0.5, 1, 0.5],
    brightness: 10,
  },
  idle: {
    base: [0.2, 0.18, 0.12],
    glow: [0.2, 0.16, 0.06],
    marker: [1, 0.85, 0.4],
    brightness: 8,
  },
  error: {
    base: [0.22, 0.08, 0.08],
    glow: [0.25, 0.06, 0.06],
    marker: [1, 0.4, 0.4],
    brightness: 7,
  },
  stopped: {
    base: [0.14, 0.14, 0.14],
    glow: [0.08, 0.08, 0.08],
    marker: [0.6, 0.6, 0.6],
    brightness: 5,
  },
  creating: {
    base: [0.12, 0.12, 0.22],
    glow: [0.08, 0.12, 0.25],
    marker: [0.5, 0.7, 1],
    brightness: 8,
  },
};

const STATUS_DOT_CSS: Record<string, string> = {
  running: "#22c55e",
  idle: "#eab308",
  error: "#ef4444",
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

// Generate a deterministic globe coordinate from a string
function stringToLocation(name: string): [number, number] {
  const h = hashCode(name);
  const lat = ((h % 1000) / 1000) * 140 - 70;
  const lng = (((h >> 10) % 1000) / 1000) * 360 - 180;
  return [lat, lng];
}

// Seeded pseudo-random
function seeded(seed: number) {
  let s = seed;
  return () => { s = (s * 16807 + 0) % 2147483647; return s / 2147483647; };
}

// Generate procedural "continent" markers — clusters of dots that form landmass shapes
function generateContinents(worldId: string, count: number): { location: [number, number]; size: number }[] {
  const rng = seeded(hashCode(worldId));
  const markers: { location: [number, number]; size: number }[] = [];

  // Generate 5-9 continent centers (more landmasses = less water)
  const numContinents = 5 + Math.floor(rng() * 5);
  const centers: [number, number][] = [];
  for (let i = 0; i < numContinents; i++) {
    centers.push([rng() * 140 - 70, rng() * 360 - 180]);
  }

  // Scatter dots around each center — wider spread, more overlap between continents
  const dotsPerContinent = Math.floor(count / numContinents);
  for (const [cLat, cLng] of centers) {
    const spread = 25 + rng() * 35; // wider continents
    for (let j = 0; j < dotsPerContinent; j++) {
      const angle = rng() * Math.PI * 2;
      const dist = (rng() + rng() + rng()) / 3 * spread; // smoother bell curve, tighter core
      const lat = cLat + Math.cos(angle) * dist;
      const lng = cLng + Math.sin(angle) * dist;
      markers.push({
        location: [Math.max(-85, Math.min(85, lat)), ((lng + 180) % 360) - 180],
        size: 0.006 + rng() * 0.012,
      });
    }
  }

  return markers;
}


export function Planet({ world, index, onClick, onEnter, isSelected, compact, hideLabels }: PlanetProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const globeRef = useRef<any>(null);
  const phiRef = useRef(hashCode(world.id) % 628 / 100); // unique starting angle
  const config = STATUS_CONFIG[world.status] ?? STATUS_CONFIG.stopped;
  const name = extractName(world.id);
  const [isMobile, setIsMobile] = useState(false);

  useEffect(() => {
    const check = () => setIsMobile(window.innerWidth < 768);
    check();
    window.addEventListener("resize", check);
    return () => window.removeEventListener("resize", check);
  }, []);

  const size = compact ? 140 : isMobile ? 140 : 200;

  const wrapperRef = useRef<HTMLDivElement>(null);
  const glowRefs = useRef<Map<string, HTMLDivElement>>(new Map());
  const selectedRef = useRef(isSelected);
  selectedRef.current = isSelected;

  useEffect(() => {
    if (!canvasRef.current) return;

    // Scale dot count to globe size for consistent density
    const dotCount = Math.round(size * 5);
    const continentMarkers = generateContinents(world.id, dotCount);

    // Agent markers (slightly larger, on top)
    const agentMarkers = world.agents.map((a) => ({
      location: stringToLocation(a.name),
      size: 0.03,
    }));

    const globe = createGlobe(canvasRef.current, {
      devicePixelRatio: 2,
      width: size * 2,
      height: size * 2,
      phi: phiRef.current,
      theta: 0.15 + index * 0.3,
      dark: 1,
      diffuse: 1.2 + (hashCode(world.id) % 10) * 0.08,
      mapSamples: 0,        // disable Earth map entirely
      mapBrightness: 0,
      mapBaseBrightness: 0,
      baseColor: config.base,
      markerColor: config.marker,
      glowColor: config.glow,
      markers: [...continentMarkers, ...agentMarkers],
      markerElevation: 0,
    });
    globeRef.current = globe;

    // Auto-rotate with gentle oscillation when selected
    let raf: number;
    let frame = 0;
    const spin = () => {
      frame++;
      const sel = selectedRef.current;

      // Throttle non-selected compact planets to every 3rd frame for performance
      if (compact && !sel && frame % 3 !== 0) {
        raf = requestAnimationFrame(spin);
        return;
      }

      phiRef.current += sel ? 0.005 : (compact ? 0.006 : 0.002);

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

  const GAP = isSelected ? 40 : 12;

  return (
    <div
      onClick={onClick}
      role="button"
      tabIndex={0}
      className="relative flex items-center justify-center focus:outline-none cursor-pointer will-change-transform"
      style={{ width: size, height: size }}
    >
      {/* ── Globe (layout anchor) ── */}
      <div
        ref={wrapperRef}
        className="will-change-[transform,filter]"
        style={{
          width: size,
          height: size,
          transform: `scale(${isSelected ? (isMobile ? 1.3 : (compact ? 1.5 : 1.8)) : (isMobile ? 0.7 : (compact ? 0.9 : 0.85))})`,
          filter: isSelected
            ? `brightness(1.2) drop-shadow(0 0 28px ${STATUS_DOT_CSS[world.status]}60)`
            : `brightness(1) drop-shadow(0 0 12px ${STATUS_DOT_CSS[world.status]}30)`,
          transition: "transform 0.9s cubic-bezier(0.16, 1, 0.3, 1), filter 0.9s cubic-bezier(0.16, 1, 0.3, 1)",
        }}
      >
        <canvas
          ref={canvasRef}
          style={{ width: size, height: size }}
        />
        {world.agents.map((a) => (
          <div
            key={`glow-${a.name}`}
            ref={(el) => { if (el) glowRefs.current.set(a.name, el); }}
            className="absolute pointer-events-none"
            style={{
              width: 28, height: 28,
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

      {/* ── Name (absolute, above globe) ── */}
      {!hideLabels && (
      <p
        className="absolute left-1/2 -translate-x-1/2 whitespace-nowrap font-heading tracking-wider text-center pointer-events-none"
        style={{
          bottom: `calc(100% + ${GAP}px)`,
          color: isSelected ? "rgba(255,255,255,0.95)" : "rgba(255,255,255,0.5)",
          fontSize: isSelected ? "1.15rem" : "0.8rem",
          letterSpacing: isSelected ? "0.12em" : "0.05em",
          opacity: isSelected ? 1 : 0.6,
          transition: "bottom 0.9s cubic-bezier(0.16, 1, 0.3, 1), opacity 0.7s ease-out, color 0.7s ease-out, font-size 0.7s ease-out, letter-spacing 0.7s ease-out",
        }}
      >
        {name}
      </p>
      )}

      {/* ── Status (absolute, below globe) ── */}
      {!hideLabels && (
      <div
        className="absolute left-1/2 -translate-x-1/2 flex items-center gap-2 whitespace-nowrap pointer-events-none"
        style={{
          top: `calc(100% + ${GAP}px)`,
          opacity: isSelected ? 1 : 0.5,
          transition: "top 0.9s cubic-bezier(0.16, 1, 0.3, 1), opacity 0.7s ease-out",
        }}
      >
        <div className="relative">
          <div
            className="w-1.5 h-1.5 rounded-full"
            style={{
              backgroundColor: STATUS_DOT_CSS[world.status],
              boxShadow: isSelected ? `0 0 8px ${STATUS_DOT_CSS[world.status]}` : "none",
              transition: "box-shadow 0.7s ease-out",
            }}
          />
          {world.status === "running" && (
            <div
              className="absolute inset-0 w-1.5 h-1.5 rounded-full animate-ping"
              style={{ backgroundColor: STATUS_DOT_CSS[world.status], opacity: 0.6 }}
            />
          )}
        </div>
        <span className="font-mono text-[10px] uppercase tracking-widest text-[rgba(255,255,255,0.3)]">
          {world.status}
        </span>
      </div>
      )}
    </div>
  );
}
