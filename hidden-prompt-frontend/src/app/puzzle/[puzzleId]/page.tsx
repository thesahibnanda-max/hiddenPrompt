"use client";

import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { usePuzzleDetail } from "@/hooks/usePuzzleDetail";
import { SiteHeader } from "@/components/chrome/SiteHeader";
import { ErrorFallback } from "@/components/feedback/ErrorFallback";
import { ChatThread } from "@/components/puzzle-detail/ChatThread";
import { GuessInput } from "@/components/puzzle-detail/GuessInput";
import { GuessScoreCard } from "@/components/puzzle-detail/GuessScoreCard";
import { WinCelebration } from "@/components/puzzle-detail/WinCelebration";
import { HiddenPromptReveal } from "@/components/puzzle-detail/HiddenPromptReveal";
import { AlreadySolvedBanner } from "@/components/puzzle-detail/AlreadySolvedBanner";
import { Skeleton } from "@/components/ui/skeleton";
import { setPuzzleOutcome } from "@/lib/utils/puzzle-outcomes";
import { howlerManager } from "@/lib/audio/howler-manager";

const UNMUTE_HINT_KEY = "hidden-prompt-unmute-hint-shown";

export default function PuzzleDetailPage() {
  const params = useParams<{ puzzleId: string }>();
  const puzzleId = params.puzzleId;

  const { details, setDetails, loading, notFound, refresh } = usePuzzleDetail(puzzleId);
  const [alreadySolved, setAlreadySolved] = useState(false);

  useEffect(() => {
    if (loading || notFound || !details) return;
    // Deliberately NOT gated on howlerManager.isStarted() - that only
    // tracks whether .play() was ever called, not whether the browser
    // actually let it through audibly. Since navigating here at all
    // requires a click (itself a qualifying gesture), isStarted() is
    // almost always already true by the time this runs even when
    // playback silently failed (the exact Incognito failure mode this
    // prompt exists for) - a one-time-ever toast is cheap enough that a
    // little redundancy when audio genuinely is playing is worth the
    // guarantee that it still shows when it silently isn't.
    if (localStorage.getItem(UNMUTE_HINT_KEY)) return;

    localStorage.setItem(UNMUTE_HINT_KEY, "1");
    toast("🔊 There's music for this — press play to hear it.", {
      action: {
        label: "Play",
        onClick: () => howlerManager.start(),
      },
    });
    // Only ever needs to fire once, the first time real content shows up -
    // re-checking on every `details` update (e.g. after each guess) would
    // just be noise since the flag is already set by then.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [loading, notFound, !!details]);

  return (
    <div className="flex min-h-screen flex-col">
      <SiteHeader />
      <main className="mx-auto flex w-full max-w-3xl flex-1 flex-col gap-5 px-4 py-6 sm:px-6 lg:px-8 3xl:max-w-4xl 3xl:px-24 3xl:py-10 tv:px-40">
        {loading && (
          <div className="flex flex-col gap-4">
            <Skeleton className="h-10 w-2/3" />
            <Skeleton className="h-24 w-full" />
            <Skeleton className="h-24 w-3/4" />
          </div>
        )}

        {!loading && notFound && (
          <ErrorFallback message="That puzzle's gone, fake, or someone else's. Back to the list?" />
        )}

        {!loading && !notFound && details && (
          <>
            <div className="card-panel sticky top-[68px] z-10 p-4 3xl:top-[84px] 3xl:p-6">
              <span className="text-[10px] uppercase tracking-widest text-chrome-mid">The AI said</span>
              <p className="chrome-text break-words text-xl font-bold leading-snug 3xl:text-3xl">
                &ldquo;{details.canonical_answer}&rdquo;
              </p>
            </div>

            <ChatThread messages={details.messages} />

            {details.latest_guess_metrics && <GuessScoreCard metrics={details.latest_guess_metrics} />}

            {details.user_win_status ? (
              <>
                <WinCelebration
                  finalScore={
                    details.latest_guess_metrics?.final_similarity_score ??
                    details.max_intent_similarity_percentage
                  }
                  isFloatPrecision={!!details.latest_guess_metrics}
                />
                {details.hidden_prompt && <HiddenPromptReveal hiddenPrompt={details.hidden_prompt} />}
              </>
            ) : alreadySolved ? (
              <AlreadySolvedBanner />
            ) : (
              <GuessInput
                puzzleId={puzzleId}
                locked={false}
                onResult={(newDetails) => {
                  setDetails(newDetails);
                  setPuzzleOutcome(puzzleId, newDetails.user_win_status);
                }}
                onAlreadySolved={() => {
                  setAlreadySolved(true);
                  refresh();
                }}
              />
            )}
          </>
        )}
      </main>
    </div>
  );
}
