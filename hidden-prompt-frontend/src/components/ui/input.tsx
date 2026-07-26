import * as React from "react";
import { cn } from "@/lib/utils/cn";

export type InputProps = React.InputHTMLAttributes<HTMLInputElement>;

const Input = React.forwardRef<HTMLInputElement, InputProps>(({ className, type, ...props }, ref) => {
  return (
    <input
      type={type}
      ref={ref}
      className={cn(
        "flex h-11 w-full rounded-md border border-white/15 bg-white/5 px-4 py-2 text-sm text-chrome-light placeholder:text-chrome-mid/60 transition-colors focus-visible:border-neon-cyan/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-neon-cyan/40 disabled:cursor-not-allowed disabled:opacity-50 3xl:h-14 3xl:text-base tv:h-16 tv:text-lg",
        className,
      )}
      {...props}
    />
  );
});
Input.displayName = "Input";

export { Input };
