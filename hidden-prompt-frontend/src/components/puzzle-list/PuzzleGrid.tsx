import type { PuzzleElement } from "@/lib/api/types";
import { PuzzleCard } from "./PuzzleCard";
import { EmptyState } from "./EmptyState";
import { Skeleton } from "@/components/ui/skeleton";

export function PuzzleGrid({ puzzles, loading }: { puzzles: PuzzleElement[]; loading: boolean }) {
  if (loading) {
    return (
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 3xl:grid-cols-4">
        {Array.from({ length: 6 }).map((_, i) => (
          <Skeleton key={i} className="h-32 w-full" />
        ))}
      </div>
    );
  }

  if (puzzles.length === 0) {
    return <EmptyState />;
  }

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 3xl:grid-cols-4">
      {puzzles.map((p, i) => (
        <PuzzleCard key={p.puzzle_id} puzzle={p} index={i} />
      ))}
    </div>
  );
}
