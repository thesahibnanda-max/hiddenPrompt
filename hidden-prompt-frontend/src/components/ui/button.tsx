import * as React from "react";
import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "@/lib/utils/cn";

const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-semibold tracking-wide transition-all duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-neon-cyan focus-visible:ring-offset-2 focus-visible:ring-offset-void disabled:pointer-events-none disabled:opacity-40 3xl:text-base 3xl:px-8 3xl:py-4 tv:text-lg tv:px-10 tv:py-5",
  {
    variants: {
      variant: {
        neon: "bg-neon-magenta text-white shadow-neon-magenta hover:brightness-110 active:scale-[0.98]",
        cyan: "bg-neon-cyan text-void shadow-neon hover:brightness-110 active:scale-[0.98]",
        outline:
          "border border-neon-cyan/50 bg-transparent text-chrome-light hover:bg-neon-cyan/10 active:scale-[0.98]",
        ghost: "bg-transparent text-chrome-light hover:bg-white/5",
        destructive: "bg-destructive text-destructive-foreground hover:brightness-110",
        link: "text-neon-cyan underline-offset-4 hover:underline",
      },
      size: {
        default: "h-10 px-5 py-2",
        sm: "h-8 px-3 text-xs",
        lg: "h-12 px-8 text-base",
        icon: "h-10 w-10",
      },
    },
    defaultVariants: {
      variant: "neon",
      size: "default",
    },
  },
);

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean;
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : "button";
    return (
      <Comp className={cn(buttonVariants({ variant, size, className }))} ref={ref} {...props} />
    );
  },
);
Button.displayName = "Button";

export { Button, buttonVariants };
