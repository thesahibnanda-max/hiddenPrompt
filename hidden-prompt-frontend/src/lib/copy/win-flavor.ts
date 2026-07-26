import { pick } from "./pick";

export const WIN_HEADLINES = [
  "YOU CRACKED IT",
  "CASE CLOSED, DETECTIVE",
  "GAME RECOGNIZES GAME",
];

export function winHeadline(): string {
  return pick(WIN_HEADLINES);
}

export function winSubline(score: string): string {
  return pick([
    `You passed with ${score} marks! Somewhere, a caveman is taking notes.`,
    `Final score: ${score}. Frame this moment, it will not happen often.`,
    `${score} marks. Certified genius, at least for today.`,
  ]);
}
