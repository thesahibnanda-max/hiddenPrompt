"use client";

import { useState } from "react";
import { Loader2, Send } from "lucide-react";
import { Textarea } from "@/components/ui/textarea";
import { Button } from "@/components/ui/button";
import { RateLimitBanner } from "@/components/feedback/RateLimitBanner";
import { submitGuess } from "@/lib/api/endpoints";
import { ApiError, RateLimitError } from "@/lib/api/errors";
import { handleUnauthorized } from "@/lib/api/handle-unauthorized";
import { genericFallback } from "@/lib/copy/error-flavor";
import type { PuzzleDetails } from "@/lib/api/types";

export function GuessInput({
  puzzleId,
  locked,
  onResult,
  onAlreadySolved,
}: {
  puzzleId: string;
  locked: boolean;
  onResult: (details: PuzzleDetails) => void;
  onAlreadySolved: () => void;
}) {
  const [value, setValue] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [rateLimit, setRateLimit] = useState<RateLimitError | null>(null);

  const submit = async () => {
    const trimmed = value.trim();
    if (!trimmed) {
      setError("You have to actually type something. We can't read minds (yet).");
      return;
    }
    setError(null);
    setLoading(true);
    try {
      const details = await submitGuess({ puzzle_id: puzzleId, user_prompt: trimmed });
      onResult(details);
      setValue("");
    } catch (err) {
      if (handleUnauthorized(err)) return;
      if (err instanceof RateLimitError) {
        setRateLimit(err);
      } else if (err instanceof ApiError && err.statusCode === 409) {
        onAlreadySolved();
      } else if (err instanceof ApiError) {
        setError(err.showAsIs ? err.body.error_message : genericFallback());
      } else {
        setError(genericFallback());
      }
    } finally {
      setLoading(false);
    }
  };

  const onKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      submit();
    }
  };

  if (locked) {
    return (
      <Textarea
        disabled
        placeholder="Puzzle solved — nothing left to guess."
        className="opacity-50"
      />
    );
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-end gap-2">
        <Textarea
          value={value}
          onChange={(e) => setValue(e.target.value)}
          onKeyDown={onKeyDown}
          disabled={loading || !!rateLimit}
          placeholder="What question do you think produced that answer?"
          className="min-h-12"
        />
        <Button
          onClick={submit}
          disabled={loading || !!rateLimit}
          size="icon"
          aria-label="Submit guess"
        >
          {loading ? <Loader2 size={16} className="animate-spin" /> : <Send size={16} />}
        </Button>
      </div>
      {error && <p className="text-sm text-destructive">{error}</p>}
      {rateLimit && (
        <RateLimitBanner retryAfterSeconds={rateLimit.retryAfterSeconds} onDone={() => setRateLimit(null)} />
      )}
    </div>
  );
}
