"use client";

import { useEffect } from "react";
import { TimerReset } from "lucide-react";
import { useCountdown } from "@/hooks/useCountdown";
import { rateLimitCopy } from "@/lib/copy/error-flavor";

export function RateLimitBanner({
  retryAfterSeconds,
  onDone,
}: {
  retryAfterSeconds: number;
  onDone?: () => void;
}) {
  const { seconds, done } = useCountdown(retryAfterSeconds);

  useEffect(() => {
    if (done) onDone?.();
  }, [done, onDone]);

  return (
    <div className="flex items-center gap-2 rounded-md border border-neon-amber/40 bg-neon-amber/10 px-4 py-3 text-sm text-neon-amber">
      <TimerReset size={16} className="shrink-0" />
      <span>{rateLimitCopy(seconds)}</span>
    </div>
  );
}
