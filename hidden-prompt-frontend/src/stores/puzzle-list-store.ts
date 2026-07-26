import { create } from "zustand";
import type { PuzzleElement } from "@/lib/api/types";

/**
 * Puzzle list state lives in a store, not component-local useState, so it
 * survives a remount of whatever component is currently rendering it. A
 * plain useState + prop-callback pattern here was found to silently drop
 * updates: if the page remounts between a button's render and its async
 * action completing (observed happening for dashboard/page.tsx around the
 * /verify -> /dashboard transition), the callback closure still points at
 * the dead, unmounted instance's setter - React 18 makes that a silent
 * no-op, so the visible (new) instance never learns about the update. A
 * store is a stable singleton outside any component's lifecycle, so a
 * remounted consumer just reads the current value fresh instead of
 * depending on which instance happened to receive the callback.
 */
interface PuzzleListState {
  puzzles: PuzzleElement[];
  setPuzzles: (puzzles: PuzzleElement[]) => void;
}

export const usePuzzleListStore = create<PuzzleListState>()((set) => ({
  puzzles: [],
  setPuzzles: (puzzles) => set({ puzzles }),
}));
