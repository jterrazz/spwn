"use client";

import { useEffect, useState } from "react";

import { Particles } from "@/components/ui/particles";

export function Stars() {
  const [isMobile, setIsMobile] = useState(false);

  useEffect(() => {
    setIsMobile(window.innerWidth < 768);
  }, []);

  return (
    <Particles
      className="fixed inset-0 z-0 opacity-35 md:opacity-55"
      color="#ffffff"
      ease={isMobile ? 40 : 50}
      quantity={isMobile ? 60 : 100}
      size={0.5}
      staticity={isMobile ? 25 : 35}
      vx={isMobile ? 0.1 : 0.04}
      vy={isMobile ? 0.05 : 0.02}
    />
  );
}
