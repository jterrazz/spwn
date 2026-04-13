"use client";

import { useEffect } from "react";

interface ShortcutHandlers {
  onSearch?: () => void;
  onSpawnWorld?: () => void;
  onEscape?: () => void;
}

/**
 * Global keyboard shortcut handler.
 * - Cmd+K / Ctrl+K → onSearch (focus search / command palette)
 * - Cmd+N / Ctrl+N → onSpawnWorld
 * - Escape → onEscape (close dialogs)
 */
export function useKeyboardShortcuts(handlers: ShortcutHandlers) {
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      const meta = e.metaKey || e.ctrlKey;

      // Don't trigger if user is typing in an input
      const tag = (e.target as HTMLElement)?.tagName;
      const isInput = tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT";

      if (meta && e.key === "k") {
        e.preventDefault();
        handlers.onSearch?.();
        return;
      }

      if (meta && e.key === "n") {
        if (!isInput) {
          e.preventDefault();
          handlers.onSpawnWorld?.();
        }
        return;
      }

      if (e.key === "Escape") {
        handlers.onEscape?.();
        return;
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [handlers]);
}
