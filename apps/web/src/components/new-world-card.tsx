"use client";

import type { CSSProperties } from "react";
import { Planet as PlanetGlobe } from "@/components/planet";
import type { World } from "@/lib/types";

// The placeholder "new world" record PlanetGlobe checks for via its id.
// When id === "w-new-00000" it renders a centered "+" glyph instead of
// a planet surface, and tints the orb muted.
const PLACEHOLDER_WORLD: World = {
  id: "w-new-00000",
  config: "default",
  agent: "",
  agents: [],
  status: "stopped",
  created_at: "",
};

interface NewWorldCardProps {
  onClick: () => void;
  /** Visual dim level - carousel uses 0.5/0.2 based on selection state,
   *  empty state uses 0.4. */
  opacity?: number;
  /** Scale override for the "dim and shrink when another planet is
   *  selected" carousel behavior. */
  scale?: number;
  /** Extra style overrides (transitions, margin, etc). */
  style?: CSSProperties;
  /** "creating" places the card inline with real worlds in the carousel,
   *  "stopped" is used for the standalone empty state - the difference
   *  only affects how PlanetGlobe tints the orb. */
  tint?: "creating" | "stopped";
}

/**
 * Shared "create a new world" card - used both inline at the end of the
 * carousel AND in the empty state. Wraps the placeholder PlanetGlobe in
 * the exact same container, animations, and typography as every other
 * world card, so the create affordance feels native to the list.
 */
export function NewWorldCard({
  onClick,
  opacity = 0.5,
  scale = 1,
  style,
  tint = "stopped",
}: NewWorldCardProps) {
  const world: World = tint === "creating"
    ? { ...PLACEHOLDER_WORLD, status: "creating" }
    : PLACEHOLDER_WORLD;

  return (
    <div
      className="group/new-world relative flex flex-col items-center shrink-0 cursor-pointer"
      style={{
        opacity,
        transform: `scale(${scale})`,
        transition:
          "opacity 320ms cubic-bezier(0.32, 0.72, 0.24, 1), " +
          "transform 420ms cubic-bezier(0.32, 0.72, 0.24, 1)",
        ...style,
      }}
      onMouseEnter={(e) => {
        e.currentTarget.style.opacity = "1";
        e.currentTarget.style.transform = `scale(${scale * 1.08})`;
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.opacity = String(opacity);
        e.currentTarget.style.transform = `scale(${scale})`;
      }}
      onClick={onClick}
    >
      {/* Soft colored halo that blooms on hover */}
      <span
        aria-hidden
        className="pointer-events-none absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 rounded-full blur-2xl opacity-0 group-hover/new-world:opacity-100 transition-opacity duration-500 ease-out"
        style={{
          width: 180,
          height: 180,
          background:
            "radial-gradient(circle, rgba(255,255,255,0.18) 0%, rgba(255,255,255,0.04) 45%, rgba(255,255,255,0) 75%)",
        }}
      />
      <PlanetGlobe
        world={world}
        index={0}
        isSelected={false}
        onClick={onClick}
        compact
        hideLabels
      />
    </div>
  );
}
