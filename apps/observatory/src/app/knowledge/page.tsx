"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";

export default function KnowledgeRedirectPage() {
  const router = useRouter();
  useEffect(() => {
    router.replace("/");
  }, [router]);
  return (
    <div className="p-8 text-muted-foreground/40 text-sm">
      Redirecting... Knowledge is now per-world.
    </div>
  );
}
