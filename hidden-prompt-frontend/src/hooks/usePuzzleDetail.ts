"use client";

import { useCallback, useEffect, useState } from "react";
import { getPuzzleDetail } from "@/lib/api/endpoints";
import { handleUnauthorized } from "@/lib/api/handle-unauthorized";
import { ApiError } from "@/lib/api/errors";
import { setPuzzleOutcome } from "@/lib/utils/puzzle-outcomes";
import { usePuzzleDetailStore } from "@/stores/puzzle-detail-store";
import type { PuzzleDetails } from "@/lib/api/types";

export function usePuzzleDetail(puzzleId: string) {
  const details = usePuzzleDetailStore((s) => s.detailsByPuzzleId[puzzleId] ?? null);
  const setDetailsInStore = usePuzzleDetailStore((s) => s.setDetails);
  const setDetails = useCallback(
    (data: PuzzleDetails) => setDetailsInStore(puzzleId, data),
    [puzzleId, setDetailsInStore],
  );

  // detailsByPuzzleId is a session-lifetime store, so returning to a
  // puzzle already viewed this session has its last-known state sitting
  // here - only block the UI when there's genuinely nothing cached for
  // this id yet. A cache hit still gets a silent background refresh below
  // (stale-while-revalidate - a puzzle's state can legitimately change,
  // e.g. after winning, so this is never a permanent cache).
  const [loading, setLoading] = useState(details === null);
  const [notFound, setNotFound] = useState(false);

  const refresh = useCallback(async () => {
    if (!details) setLoading(true);
    setNotFound(false);
    try {
      const data = await getPuzzleDetail(puzzleId);
      setDetails(data);
      setPuzzleOutcome(puzzleId, data.user_win_status);
    } catch (err) {
      if (handleUnauthorized(err)) return;
      if (err instanceof ApiError && (err.statusCode === 404 || err.statusCode === 400)) {
        setNotFound(true);
      }
    } finally {
      setLoading(false);
    }
  }, [puzzleId, setDetails, details]);

  useEffect(() => {
    refresh();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [puzzleId]);

  return { details, setDetails, loading, notFound, refresh };
}
