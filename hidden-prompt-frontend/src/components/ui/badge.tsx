import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "@/lib/utils/cn";

const badgeVariants = cva(
  "inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold tracking-wide transition-colors 3xl:text-sm",
  {
    variants: {
      variant: {
        won: "border-neon-cyan/50 bg-neon-cyan/10 text-neon-cyan shadow-neon",
        unsolved: "border-neon-magenta/50 bg-neon-magenta/10 text-neon-magenta",
        neutral: "border-white/20 bg-white/5 text-chrome-mid",
        warn: "border-neon-amber/50 bg-neon-amber/10 text-neon-amber",
      },
    },
    defaultVariants: { variant: "neutral" },
  },
);

export interface BadgeProps extends React.HTMLAttributes<HTMLDivElement>, VariantProps<typeof badgeVariants> {}

function Badge({ className, variant, ...props }: BadgeProps) {
  return <div className={cn(badgeVariants({ variant }), className)} {...props} />;
}

export { Badge, badgeVariants };
