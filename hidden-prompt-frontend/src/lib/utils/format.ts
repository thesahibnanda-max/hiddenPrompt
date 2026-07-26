export function formatTimestamp(iso: string): string {
  try {
    return new Date(iso).toLocaleString(undefined, {
      hour: "2-digit",
      minute: "2-digit",
      month: "short",
      day: "numeric",
    });
  } catch {
    return iso;
  }
}

/**
 * final_similarity_score (from a fresh guess response) is a float — show one
 * decimal. max_intent_similarity_percentage (the persisted fallback shown on
 * a later plain GET, where latest_guess_metrics is absent) is an int — show
 * as-is. Same underlying winning score, two precisions, one formatter.
 */
export function formatScore(score: number, isFloatPrecision: boolean): string {
  return isFloatPrecision ? score.toFixed(1) : Math.round(score).toString();
}
