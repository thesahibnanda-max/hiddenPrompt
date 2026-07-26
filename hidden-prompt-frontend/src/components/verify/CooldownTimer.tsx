"use client";

import { useCountdown } from "@/hooks/useCountdown";

export function CooldownTimer({
  seconds,
  label = "Try again in",
}: {
  seconds: number;
  label?: string;
}) {
  const { formatted, done } = useCountdown(seconds);
  if (done) return null;
  return (
    <span className="text-xs text-neon-amber">
      {label} {formatted}
    </span>
  );
}
