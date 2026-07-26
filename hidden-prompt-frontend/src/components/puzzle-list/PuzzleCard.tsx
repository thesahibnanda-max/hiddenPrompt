"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { motion } from "framer-motion";
import { Loader2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { getPuzzleOutcome } from "@/lib/utils/puzzle-outcomes";
import type { PuzzleElement } from "@/lib/api/types";

// isNavigating is deliberately local, plain React state - not tied to
// Next.js routing/Suspense timing at all. It exists purely to paint
// feedback in the same tick as the click, regardless of how long the
// actual navigation takes (dev-mode compile, PageTransition's exit
// animation, the API fetch) - "did my click register" should never be in
// doubt. A safety timeout resets it if navigation never completes (e.g. a
// network error), so the card doesn't stay spinner-locked forever.
//
// Kept comfortably above realistic worst-case backend latency (was 8s,
// which turned out to be *shorter* than an intermittent backend stall
// this was meant to guard against - the safety net itself was firing
// before real navigation finished, hiding the spinner while the app sat
// idle for several more seconds). Now that the underlying stall is fixed
// at the source, this should rarely if ever actually fire in practice.
const NAVIGATION_TIMEOUT_MS = 20000;

export function PuzzleCard({ puzzle, index }: { puzzle: PuzzleElement; index: number }) {
  const outcome = getPuzzleOutcome(puzzle.puzzle_id);
  const [isNavigating, setIsNavigating] = useState(false);
  // Tracks the pending safety-timeout id so a rapid re-click clears the
  // previous one instead of letting timers stack up, each independently
  // trying to reset isNavigating later.
  const navigationTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    return () => {
      if (navigationTimeoutRef.current) clearTimeout(navigationTimeoutRef.current);
    };
  }, []);

  return (
    <motion.div
      initial={{ opacity: 0, y: 16, scale: 0.98 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      transition={{ duration: 0.3, delay: Math.min(index * 0.04, 0.4) }}
      className="relative"
    >
      <Link
        href={`/puzzle/${puzzle.puzzle_id}`}
        onClick={() => {
          setIsNavigating(true);
          if (navigationTimeoutRef.current) clearTimeout(navigationTimeoutRef.current);
          navigationTimeoutRef.current = setTimeout(() => setIsNavigating(false), NAVIGATION_TIMEOUT_MS);
        }}
        className="card-panel group flex h-full flex-col gap-3 p-5 transition-all hover:border-neon-cyan/50 hover:shadow-neon 3xl:p-8"
      >
        <div className="flex items-start justify-between gap-2">
          <span className="text-[10px] uppercase tracking-widest text-chrome-mid">The answer was</span>
          {outcome === true && <Badge variant="won">Cracked it</Badge>}
          {outcome === false && <Badge variant="unsolved">Still cooking</Badge>}
          {outcome === undefined && <Badge variant="neutral">Tap to play</Badge>}
        </div>
        <p className="chrome-text line-clamp-3 text-lg font-bold leading-snug 3xl:text-2xl">
          &ldquo;{puzzle.canonical_answer}&rdquo;
        </p>
      </Link>
      {isNavigating && (
        <div className="absolute inset-0 flex items-center justify-center rounded-lg bg-void/70 backdrop-blur-sm">
          <Loader2 className="animate-spin text-neon-cyan" size={28} />
        </div>
      )}
    </motion.div>
  );
}
