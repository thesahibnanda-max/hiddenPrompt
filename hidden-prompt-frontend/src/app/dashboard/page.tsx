"use client";

import { usePuzzles } from "@/hooks/usePuzzles";
import { PuzzleGrid } from "@/components/puzzle-list/PuzzleGrid";
import { NewPuzzleButton } from "@/components/puzzle-list/NewPuzzleButton";

export default function DashboardPage() {
  const { puzzles, setPuzzles, loading } = usePuzzles();

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="chrome-text text-2xl font-black tracking-wide 3xl:text-4xl">YOUR PUZZLES</h1>
          <p className="text-sm text-chrome-mid">Pick one. Regret nothing.</p>
        </div>
        <NewPuzzleButton onCreated={setPuzzles} />
      </div>
      <PuzzleGrid puzzles={puzzles} loading={loading} />
    </div>
  );
}
