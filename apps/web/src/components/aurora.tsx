"use client";

import { cn } from "@/lib/utils";

export function Aurora({ className }: { className?: string }) {
  return (
    <div
      aria-hidden
      className={cn(
        "pointer-events-none fixed inset-0 z-[1] overflow-hidden [transform:translateZ(0)]",
        className,
      )}
    >
      <div
        className={cn(
          "absolute -inset-[10px]",
          "opacity-[0.06] blur-[18px] will-change-transform",
          "dark:invert-0 invert filter",

          // Aurora gradient - slightly varied angles to break parallelism
          "[--aurora:repeating-linear-gradient(100deg,rgba(255,255,255,0.15)_10%,rgba(180,180,180,0.1)_15%,rgba(220,220,220,0.12)_20%,rgba(150,150,150,0.08)_25%,rgba(200,200,200,0.1)_30%)]",
          "[--aurora-b:repeating-linear-gradient(115deg,rgba(255,255,255,0.1)_8%,rgba(200,200,200,0.06)_14%,rgba(240,240,240,0.08)_22%,rgba(160,160,160,0.05)_28%,rgba(210,210,210,0.07)_34%)]",

          // Stripe pattern overlay
          "[--dark-gradient:repeating-linear-gradient(100deg,rgba(0,0,0,1)_0%,rgba(0,0,0,1)_7%,transparent_10%,transparent_12%,rgba(0,0,0,1)_16%)]",
          "[--white-gradient:repeating-linear-gradient(100deg,rgba(255,255,255,1)_0%,rgba(255,255,255,1)_7%,transparent_10%,transparent_12%,rgba(255,255,255,1)_16%)]",

          // Background composition
          "[background-image:var(--white-gradient),var(--aurora)]",
          "[background-size:300%,_200%]",
          "[background-position:50%_50%,50%_50%]",

          // Dark mode
          "dark:[background-image:var(--dark-gradient),var(--aurora)]",

          // After pseudo - second layer at different angle for depth
          "after:absolute after:inset-0 after:content-['']",
          "after:[background-image:var(--white-gradient),var(--aurora-b)]",
          "after:[background-size:250%,_180%]",
          "after:[background-attachment:fixed]",
          "after:mix-blend-difference",
          "after:dark:[background-image:var(--dark-gradient),var(--aurora-b)]",

          // Mask - simple top fade
          "[mask-image:radial-gradient(ellipse_at_50%_0%,black_10%,transparent_70%)]",

          // Animation
          "after:animate-aurora",
          "animate-aurora",
        )}
      />
    </div>
  );
}
