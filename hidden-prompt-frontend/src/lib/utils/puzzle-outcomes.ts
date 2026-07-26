// Lightweight, intentionally-imperfect win/unsolved cache for puzzle list
// badges. GET /puzzle doesn't return win status, and fetching detail for
// every card would be an N+1 request storm against a 30/min rate limit for
// a mere badge. So: only record an outcome once a puzzle has actually been
// opened (its user_win_status becomes known from GET /puzzle/detail or
// POST /puzzle/guess). Unvisited puzzles just show a neutral badge instead
// of guessing. sessionStorage (not persisted long-term) since this is a
// display nicety, not a source of truth.

const KEY = "hidden-prompt-puzzle-outcomes";

function readAll(): Record<string, boolean> {
  if (typeof window === "undefined") return {};
  try {
    return JSON.parse(sessionStorage.getItem(KEY) ?? "{}");
  } catch {
    return {};
  }
}

export function getPuzzleOutcome(puzzleId: string): boolean | undefined {
  return readAll()[puzzleId];
}

export function setPuzzleOutcome(puzzleId: string, won: boolean) {
  if (typeof window === "undefined") return;
  const all = readAll();
  all[puzzleId] = won;
  sessionStorage.setItem(KEY, JSON.stringify(all));
}
