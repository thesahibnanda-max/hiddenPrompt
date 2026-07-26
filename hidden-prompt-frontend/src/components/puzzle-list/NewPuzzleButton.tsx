"use client";

import { useState } from "react";
import dynamic from "next/dynamic";
import { toast } from "sonner";
import { Sparkles } from "lucide-react";
import { Button } from "@/components/ui/button";
import { RateLimitBanner } from "@/components/feedback/RateLimitBanner";
import { createPuzzle } from "@/lib/api/endpoints";
import { ApiError, RateLimitError } from "@/lib/api/errors";
import { handleUnauthorized } from "@/lib/api/handle-unauthorized";
import { useAuthStore } from "@/stores/auth-store";
import { genericFallback } from "@/lib/copy/error-flavor";
import type { PuzzleElement } from "@/lib/api/types";

// Dynamically imported since it (and its @radix-ui/react-dialog
// dependency) is only ever needed while a puzzle is actively being
// created - no reason to ship it in the initial dashboard bundle for
// every visit.
const NewPuzzleLoadingOverlay = dynamic(() =>
  import("./NewPuzzleLoadingOverlay").then((m) => m.NewPuzzleLoadingOverlay),
);

export function NewPuzzleButton({ onCreated }: { onCreated: (puzzles: PuzzleElement[]) => void }) {
  const setVerified = useAuthStore((s) => s.setVerified);
  const [loading, setLoading] = useState(false);
  const [rateLimit, setRateLimit] = useState<RateLimitError | null>(null);

  const onClick = async () => {
    setLoading(true);
    setRateLimit(null);
    try {
      const puzzles = await createPuzzle();
      onCreated(puzzles);
      toast.success("Fresh puzzle incoming.");
    } catch (err) {
      if (handleUnauthorized(err)) return;
      if (err instanceof RateLimitError) {
        setRateLimit(err);
      } else if (err instanceof ApiError && err.statusCode === 403) {
        // AuthGate reacts to `verified` going false on this route and
        // navigates to /verify on its own — no manual replace needed here.
        setVerified(false);
        toast.error("Whoa there — verify your email first.");
      } else if (err instanceof ApiError) {
        toast.error(err.showAsIs ? err.body.error_message : genericFallback());
      } else {
        toast.error(genericFallback());
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex flex-col items-end gap-2">
      <Button onClick={onClick} disabled={loading || !!rateLimit} variant="neon">
        <Sparkles size={16} />
        New Puzzle
      </Button>
      {rateLimit && (
        <RateLimitBanner retryAfterSeconds={rateLimit.retryAfterSeconds} onDone={() => setRateLimit(null)} />
      )}
      <NewPuzzleLoadingOverlay open={loading} />
    </div>
  );
}
