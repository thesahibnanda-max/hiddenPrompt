"use client";

import * as React from "react";
import { cn } from "@/lib/utils/cn";

interface OtpInputProps {
  value: string;
  onChange: (value: string) => void;
  maxLength?: number;
  shake?: boolean;
  disabled?: boolean;
}

// A single input rather than fixed per-character boxes: the backend's OTP
// is randomly 5 OR 6 characters (utils.ChooseOneRandomly(5, 6) server-side)
// with no way to know the length in advance, so a fixed-slot box UI always
// left a dangling empty box for 5-character codes. A plain input handles
// either length correctly with no visual mismatch.
export function OtpInput({ value, onChange, maxLength = 6, shake, disabled }: OtpInputProps) {
  return (
    <input
      type="text"
      inputMode="text"
      autoComplete="one-time-code"
      value={value}
      disabled={disabled}
      maxLength={maxLength}
      onChange={(e) => onChange(e.target.value.replace(/\s/g, ""))}
      className={cn(
        // NOT `uppercase` - the OTP itself is a case-sensitive mixed-case
        // string (utils.GenerateRandomString server-side), forcing case
        // here would silently corrupt whatever the user actually typed.
        "h-16 w-56 rounded-md border border-white/15 bg-white/5 text-center font-mono text-2xl font-bold tracking-[0.5em] text-chrome-light transition-all focus-visible:border-neon-cyan focus-visible:shadow-neon focus-visible:outline-none disabled:opacity-50 3xl:h-20 3xl:w-64 3xl:text-3xl",
        shake && "animate-shake",
      )}
    />
  );
}
