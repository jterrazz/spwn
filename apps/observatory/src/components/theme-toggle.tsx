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
      className="w-8 h-8 flex items-center justify-center rounded-full text-muted-foreground/30 hover:text-foreground transition-colors"
      aria-label="Toggle theme"
    >
      {isDark ? (
        <IconMoonFilled size={15} />
      ) : (
        <IconSunFilled size={15} />
      )}
    </button>
  );
}
