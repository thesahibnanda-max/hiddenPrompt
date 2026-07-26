import { cn } from "@/lib/utils/cn";

interface ProgressMeterProps {
  value: number; // 0-100
  colorClassName?: string;
  className?: string;
}

/** A themed meter bar (not Radix Progress — simpler, and we want the neon fill styling). */
export function ProgressMeter({ value, colorClassName = "bg-neon-cyan", className }: ProgressMeterProps) {
  const clamped = Math.max(0, Math.min(100, value));
  return (
    <div className={cn("h-2 w-full overflow-hidden rounded-full bg-white/10", className)}>
      <div
        className={cn("h-full rounded-full transition-all duration-500", colorClassName)}
        style={{ width: `${clamped}%` }}
      />
    </div>
  );
}
