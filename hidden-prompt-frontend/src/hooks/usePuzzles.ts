"use client";

import { useCallback, useEffect, useState } from "react";
import { getPuzzles } from "@/lib/api/endpoints";
import { handleUnauthorized } from "@/lib/api/handle-unauthorized";
import { ApiError } from "@/lib/api/errors";
import { usePuzzleListStore } from "@/stores/puzzle-list-store";
import { useAuthStore } from "@/stores/auth-store";

export function usePuzzles() {
  const puzzles = usePuzzleListStore((s) => s.puzzles);
  const setPuzzles = usePuzzleListStore((s) => s.setPuzzles);
  const setVerified = useAuthStore((s) => s.setVerified);
  // The store is a session-lifetime singleton, so a revisit to /dashboard
  // (not a full page reload) already has last time's data sitting here -
  // only block the UI on a genuinely empty/cold cache. A non-empty cache
  // still gets refreshed below, just silently (stale-while-revalidate)
  // instead of flashing the skeleton again.
  const [loading, setLoading] = useState(puzzles.length === 0);

  const refresh = useCallback(async () => {
    if (puzzles.length === 0) setLoading(true);
    try {
      const data = await getPuzzles();
      setPuzzles(data);
    } catch (err) {
      if (handleUnauthorized(err)) return;
      if (err instanceof ApiError && err.statusCode === 403) {
        // Defensive fallback only - the login response's `is_verified`
        // field should already keep this in sync. AuthGate reacts to
        // `verified` going false on this route and navigates to /verify
        // on its own, same pattern as NewPuzzleButton's 403 handling.
        setVerified(false);
      }
    } finally {
      setLoading(false);
    }
  }, [setPuzzles, setVerified, puzzles.length]);

  useEffect(() => {
    refresh();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return { puzzles, setPuzzles, loading, refresh };
}
