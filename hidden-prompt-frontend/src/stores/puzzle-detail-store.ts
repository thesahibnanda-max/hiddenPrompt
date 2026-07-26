import { create } from "zustand";
import type { PuzzleDetails } from "@/lib/api/types";

/**
 * Keyed by puzzle_id so state survives a remount of whatever component is
 * currently rendering it - same rationale as puzzle-list-store.ts. A guess
 * result landing on a useState setter bound to an already-remounted (dead)
 * page instance would otherwise be silently lost, which is much worse here
 * than for the puzzle list: it's the core gameplay action.
 */
interface PuzzleDetailState {
  detailsByPuzzleId: Record<string, PuzzleDetails>;
  setDetails: (puzzleId: string, details: PuzzleDetails) => void;
}

export const usePuzzleDetailStore = create<PuzzleDetailState>()((set) => ({
  detailsByPuzzleId: {},
  setDetails: (puzzleId, details) =>
    set((state) => ({ detailsByPuzzleId: { ...state.detailsByPuzzleId, [puzzleId]: details } })),
}));
