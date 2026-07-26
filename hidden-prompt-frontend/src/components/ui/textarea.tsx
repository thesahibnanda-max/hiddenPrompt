import * as React from "react";
import { cn } from "@/lib/utils/cn";

export type TextareaProps = React.TextareaHTMLAttributes<HTMLTextAreaElement>;

const Textarea = React.forwardRef<HTMLTextAreaElement, TextareaProps>(({ className, ...props }, ref) => {
  return (
    <textarea
      ref={ref}
      className={cn(
        "flex min-h-20 w-full resize-none rounded-md border border-white/15 bg-white/5 px-4 py-3 text-sm text-chrome-light placeholder:text-chrome-mid/60 transition-colors focus-visible:border-neon-cyan/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-neon-cyan/40 disabled:cursor-not-allowed disabled:opacity-50 3xl:text-base tv:text-lg",
        className,
      )}
      {...props}
    />
  );
});
Textarea.displayName = "Textarea";

export { Textarea };
