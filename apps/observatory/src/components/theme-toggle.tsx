"use client";

import { useTheme } from "next-themes";
import { useEffect, useState } from "react";
import { IconSunFilled, IconMoonFilled } from "@tabler/icons-react";

export function ThemeToggle() {
  const { theme, setTheme } = useTheme();
  const [mounted, setMounted] = useState(false);
  useEffect(() => setMounted(true), []);

  if (!mounted) return <div className="w-8 h-8" />;

  const isDark = theme === "dark";

  return (
    <button
      onClick={() => setTheme(isDark ? "light" : "dark")}
      className="w-8 h-8 flex items-center justify-center rounded-full bg-foreground/[0.06] dark:bg-white/[0.08] backdrop-blur-md border border-foreground/[0.08] dark:border-white/[0.1] text-muted-foreground/50 hover:text-foreground shadow-[inset_0_1px_0_rgba(255,255,255,0.12),0_1px_2px_rgba(0,0,0,0.05)] dark:shadow-[inset_0_1px_0_rgba(255,255,255,0.06),0_1px_2px_rgba(0,0,0,0.2)] transition-colors"
      aria-label="Toggle theme"
    >
      {isDark ? (
        <IconSunFilled size={14} />
      ) : (
        <IconMoonFilled size={14} />
      )}
    </button>
  );
}
