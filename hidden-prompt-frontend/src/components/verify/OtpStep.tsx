"use client";

import { motion } from "framer-motion";
import { KeyRound } from "lucide-react";
import { Button } from "@/components/ui/button";
import { OtpInput } from "@/components/ui/otp-input";
import { RateLimitBanner } from "@/components/feedback/RateLimitBanner";
import { BackToHomeLink } from "./BackToHomeLink";

export function OtpStep({
  otp,
  onOtpChange,
  onSubmit,
  onResend,
  loading,
  resendLoading,
  error,
  shake,
  rateLimitSeconds,
  onRateLimitDone,
}: {
  otp: string;
  onOtpChange: (v: string) => void;
  onSubmit: () => void;
  onResend: () => void;
  loading: boolean;
  resendLoading: boolean;
  error: string | null;
  shake: boolean;
  rateLimitSeconds: number | null;
  onRateLimitDone: () => void;
}) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      className="flex flex-col items-center gap-5 text-center"
    >
      <div className="flex h-16 w-16 items-center justify-center rounded-full bg-neon-magenta/10 text-neon-magenta 3xl:h-20 3xl:w-20">
        <KeyRound size={30} />
      </div>
      <div>
        <h2 className="chrome-text text-xl font-bold 3xl:text-2xl">Check your inbox</h2>
        <p className="mt-1 text-sm text-chrome-mid">Enter the code we just sent you.</p>
      </div>

      <OtpInput value={otp} onChange={onOtpChange} maxLength={6} shake={shake} disabled={loading} />

      {error && <p className="text-sm text-destructive">{error}</p>}
      {rateLimitSeconds !== null && (
        <RateLimitBanner retryAfterSeconds={rateLimitSeconds} onDone={onRateLimitDone} />
      )}

      <Button
        onClick={onSubmit}
        disabled={loading || otp.length < 5 || rateLimitSeconds !== null}
        size="lg"
        className="w-full"
      >
        {loading ? "Verifying..." : "Verify"}
      </Button>
      <button
        type="button"
        onClick={onResend}
        disabled={resendLoading || rateLimitSeconds !== null}
        className="text-xs text-chrome-mid underline-offset-4 hover:text-neon-cyan hover:underline disabled:opacity-40"
      >
        {resendLoading ? "Resending..." : "Resend code"}
      </button>
      <BackToHomeLink />
    </motion.div>
  );
}
