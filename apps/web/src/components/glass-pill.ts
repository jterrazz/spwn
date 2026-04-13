/**
 * Shared glass-pill styling used by every 42-pixel header element:
 * ActionButton, ExpandingSearch, DashboardHeaderStats, etc. Kept in one
 * place so the chrome can never drift between components.
 *
 * Note: classes that depend on component shape (padding, width) should
 * live at the call site — this only captures the non-geometric look
 * (border, glass bg, shadow, blur, the -1px vertical nudge to align
 * with PageHeader's title baseline).
 */
export const GLASS_PILL_CLASS =
  "rounded-full border " +
  "border-foreground/[0.08] dark:border-white/[0.1] " +
  "bg-foreground/[0.04] dark:bg-white/[0.05] backdrop-blur-md " +
  "shadow-[inset_0_1px_0_rgba(255,255,255,0.08),0_1px_2px_rgba(0,0,0,0.04)] " +
  "dark:shadow-[inset_0_1px_0_rgba(255,255,255,0.05),0_1px_2px_rgba(0,0,0,0.18)] " +
  "text-foreground/78 -translate-y-1";

/** Canonical 42px height — the height of every header pill. */
export const GLASS_PILL_HEIGHT = 42;
