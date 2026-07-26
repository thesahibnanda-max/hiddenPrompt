import { pick } from "./pick";

/**
 * Every guess surfaces all four backend metrics — cosine, LLM, embedding
 * dot product, and the blended final score — each translated into its own
 * dumb-simple joke instead of a raw number. No single narrator character;
 * each metric gets its own voice/tier pool.
 */

// dotProduct arrives as a raw (non-normalized) embedding dot product, which
// for this embedding model comes out ~[-1, 1] in practice (same shape as
// cosine similarity, just unscaled) — see dotMatrixMeterValue for the 0-100
// meter conversion these jokes are bucketed against.
export function dotMatrixFlavor(dotProduct: number): string {
  const pct = dotProduct * 100;
  if (pct >= 90) {
    return pick([
      "Vectors locked in. Perfect alignment.",
      "Your signal and the answer's signal are basically one beam.",
      "Dead-on. The matrix approves.",
    ]);
  }
  if (pct >= 70) {
    return pick([
      "Strong signal. Barely any static.",
      "Vectors pointing the same direction, mostly.",
      "Good alignment. A little drift, nothing major.",
    ]);
  }
  if (pct >= 40) {
    return pick([
      "Signal's there, but it's crossing wires.",
      "Some overlap. The dots aren't quite connecting.",
      "Half-tuned. Static in the middle.",
    ]);
  }
  return pick([
    "Vectors pointing in completely different directions.",
    "No signal. Just noise.",
    "The matrix has no idea what you're talking about.",
  ]);
}

export function vibeCheckFlavor(cosineScore: number): string {
  if (cosineScore >= 90) {
    return pick([
      "Your brain and the robot's brain are basically dating.",
      "Certified soulmates, embedding-wise.",
      "You two finish each other's vectors.",
    ]);
  }
  if (cosineScore >= 70) {
    return pick([
      "Same neighborhood, different house.",
      "Close enough to wave at from across the street.",
      "You're circling the right idea. Keep circling.",
    ]);
  }
  if (cosineScore >= 40) {
    return pick([
      "Same planet, wrong continent.",
      "Distant cousins at best.",
      "There's a vague family resemblance. Very vague.",
    ]);
  }
  return pick([
    "Different universe entirely. Bold choice.",
    "Not even the same species of thought.",
    "The vibes could not be more off.",
  ]);
}

export function mindReadingFlavor(llmScore: number, wasComputed: boolean): string {
  if (!wasComputed) {
    return pick([
      "The AI didn't even bother reading this one. Yikes.",
      "Too far off to earn a second look from the AI.",
      "The AI took one glance and moved on with its day.",
    ]);
  }
  if (llmScore >= 90) {
    return pick([
      "The AI is convinced you can read its mind.",
      "The AI wants to know your secrets.",
      "The AI is genuinely a little spooked right now.",
    ]);
  }
  if (llmScore >= 70) {
    return pick([
      "The AI gets the gist. Kind of impressed.",
      "The AI is nodding along, mostly.",
      "The AI thinks you're onto something.",
    ]);
  }
  if (llmScore >= 40) {
    return pick([
      "The AI is squinting at your guess.",
      "The AI is giving you the benefit of the doubt. Barely.",
      "The AI needs you to explain your reasoning.",
    ]);
  }
  return pick([
    "The AI has filed a restraining order against your guess.",
    "The AI would like to speak to your manager.",
    "The AI is quietly judging you right now.",
  ]);
}

export function verdictFlavor(finalScore: number, consideredWon: boolean): string {
  if (consideredWon) {
    return pick([
      "WINNER WINNER. Mic drop.",
      "Case closed. Go tell your friends.",
      "That's a win. Frame it.",
    ]);
  }
  if (finalScore >= 80) {
    return pick([
      "So close the AI can taste your victory. Try again.",
      "One more nudge and this is over.",
      "You're basically knocking on the answer's door.",
    ]);
  }
  if (finalScore >= 50) {
    return pick([
      "Mid. Painfully, aggressively mid.",
      "Not bad, not good, just... there.",
      "Right ballpark, wrong seat.",
    ]);
  }
  return pick([
    "Not even close. Godspeed.",
    "That guess and the answer have never met.",
    "Back to the drawing board. A bigger drawing board.",
  ]);
}

export const NEAR_MISS_LOSS = [
  "So close you can smell the win. 90 or bust, and you're not there yet.",
  "Just under the bar. Even the scoreboard is squinting.",
  "One good guess away from bragging rights.",
];

export function nearMissFlavor(finalScore: number, consideredWon: boolean): string | null {
  if (consideredWon) return null;
  if (finalScore >= 80 && finalScore < 90) {
    return pick(NEAR_MISS_LOSS);
  }
  return null;
}
