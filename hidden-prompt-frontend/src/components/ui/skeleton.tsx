import { cn } from "@/lib/utils/cn";

function Skeleton({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn("animate-pulse rounded-md bg-gradient-to-r from-white/5 via-white/10 to-white/5", className)}
      {...props}
    />
  );
}

export { Skeleton };
