export interface HowToPlayStep {
  title: string;
  body: string;
}

export const HOW_TO_PLAY_STEPS: HowToPlayStep[] = [
  {
    title: "There's a secret question.",
    body: "An AI already answered it. You only see the answer — like \"42\" or \"probably not.\" The question is hidden from you.",
  },
  {
    title: "Your job: guess the hidden question.",
    body: "That's the whole game. Look at the answer, then guess what question made the AI say that. Sounds easy. It's not.",
  },
  {
    title: "A computer checks your guess.",
    body: "You don't need the exact words. It just checks if your guess means the same thing as the real question. It's picky.",
  },
  {
    title: "Guess close enough, you win.",
    body: "Nail it, and the real hidden question gets shown to you. That's it. That's the win.",
  },
];
