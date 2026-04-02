"use client";

import { usePathname } from "next/navigation";
import { IconChevronRight } from "@tabler/icons-react";

function extractWorldName(id: string): string {
  const parts = id.split("-");
  return parts.length >= 2
    ? parts[1].charAt(0).toUpperCase() + parts[1].slice(1)
    : id;
}

interface Crumb {
  label: string;
  href: string;
}

export function Breadcrumbs() {
  const pathname = usePathname();

  const crumbs: Crumb[] = [{ label: "Meson", href: "/" }];

  if (pathname === "/" || pathname === "") {
    return null; // No breadcrumbs on root
  }

  if (pathname === "/architect") {
    crumbs.push({ label: "Architect", href: "/architect" });
  } else if (pathname === "/marketplace") {
    crumbs.push({ label: "Marketplace", href: "/marketplace" });
  } else if (pathname.startsWith("/world/")) {
    const parts = pathname.split("/").filter(Boolean);
    const worldId = parts[1];
    if (worldId) {
      crumbs.push({
        label: extractWorldName(worldId),
        href: `/world/${worldId}`,
      });
    }
    const agentName = parts[2];
    if (agentName) {
      crumbs.push({
        label: agentName,
        href: `/world/${worldId}/${agentName}`,
      });
    }
  } else if (pathname.startsWith("/agents/")) {
    const agentName = pathname.split("/")[2];
    crumbs.push({ label: "Agents", href: "/agents" });
    if (agentName) {
      crumbs.push({ label: agentName, href: pathname });
    }
  }

  if (crumbs.length <= 1) return null;

  return (
    <nav className="flex items-center gap-1 px-6 pt-4 pb-0">
      {crumbs.map((crumb, i) => (
        <span key={crumb.href} className="flex items-center gap-1">
          {i > 0 && (
            <IconChevronRight
              size={12}
              className="text-muted-foreground/20"
            />
          )}
          {i < crumbs.length - 1 ? (
            <a
              href={crumb.href}
              className="text-[11px] font-mono text-muted-foreground/40 hover:text-muted-foreground/70 transition-colors"
            >
              {crumb.label}
            </a>
          ) : (
            <span className="text-[11px] font-mono text-muted-foreground/60">
              {crumb.label}
            </span>
          )}
        </span>
      ))}
    </nav>
  );
}
