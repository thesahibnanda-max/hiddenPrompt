"use client";

import { Loader2 } from "lucide-react";

export function MusicLoadingState() {
  return (
    <div className="flex flex-col items-center gap-3 p-6 text-center">
      <Loader2 size={24} className="animate-spin text-neon-cyan" />
      <p className="text-xs text-chrome-mid">Loading some songs for you while you play&hellip;</p>
    </div>
  );
}
