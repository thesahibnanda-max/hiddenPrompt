import { Ghost } from "lucide-react";

export function EmptyState() {
  return (
    <div className="flex flex-col items-center gap-3 rounded-lg border border-dashed border-white/15 p-14 text-center">
      <Ghost className="text-neon-purple" size={36} />
      <h3 className="chrome-text text-lg font-bold">No puzzles yet.</h3>
      <p className="max-w-xs text-sm text-chrome-mid">
        The AI is bored and has nothing to say. Fix that with a new puzzle.
      </p>
    </div>
  );
}
