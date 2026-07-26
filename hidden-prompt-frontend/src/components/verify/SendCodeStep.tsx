"use client";

import { motion } from "framer-motion";
import { Mail } from "lucide-react";
import { Button } from "@/components/ui/button";
import { RateLimitBanner } from "@/components/feedback/RateLimitBanner";
import { BackToHomeLink } from "./BackToHomeLink";

export function SendCodeStep({
  onSend,
  loading,
  error,
  rateLimitSeconds,
  onRateLimitDone,
  alreadyVerifiedHint,
}: {
  onSend: () => void;
  loading: boolean;
  error: string | null;
  rateLimitSeconds: number | null;
  onRateLimitDone: () => void;
  alreadyVerifiedHint?: string | null;
}) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      className="flex flex-col items-center gap-5 text-center"
    >
      <div className="flex h-16 w-16 items-center justify-center rounded-full bg-neon-cyan/10 text-neon-cyan 3xl:h-20 3xl:w-20">
        <Mail size={30} />
      </div>
      <div>
        <h2 className="chrome-text text-xl font-bold 3xl:text-2xl">Prove you&apos;re a real human</h2>
        <p className="mt-1 text-sm text-chrome-mid">
          We&apos;ll email you a code. Type it back to us and you&apos;re in.
        </p>
      </div>

      {alreadyVerifiedHint && (
        <p className="text-sm text-neon-cyan">{alreadyVerifiedHint}</p>
      )}
      {error && <p className="text-sm text-destructive">{error}</p>}
      {rateLimitSeconds !== null && (
        <RateLimitBanner retryAfterSeconds={rateLimitSeconds} onDone={onRateLimitDone} />
      )}

      <Button onClick={onSend} disabled={loading || rateLimitSeconds !== null} size="lg" className="w-full">
        {loading ? "Sending..." : "Get Verified"}
      </Button>
      <BackToHomeLink />
    </motion.div>
  );
}
