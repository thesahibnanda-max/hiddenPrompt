"use client";

import { useMemo } from "react";
import { motion } from "framer-motion";
import { ProgressMeter } from "@/components/ui/progress-meter";
import {
  dotMatrixFlavor,
  vibeCheckFlavor,
  mindReadingFlavor,
  verdictFlavor,
  nearMissFlavor,
} from "@/lib/copy/guess-report-flavor";
import type { GuessMetrics } from "@/lib/api/types";

// The raw dot product isn't bounded 0-100 like the other three scores — in
// practice it comes out ~[-1, 1] for this embedding model (unit-normalized
// vectors), so scale it the same way cosine already is for the meter bar.
function dotProductToMeterValue(dotProduct: number): number {
  return Math.max(0, Math.min(100, dotProduct * 100));
}

export function GuessScoreCard({ metrics }: { metrics: GuessMetrics }) {
  const llmComputed = metrics.cosine_similarity_score >= 60;

  const rows = useMemo(
    () => [
      {
        label: "Dot Matrix",
        value: dotProductToMeterValue(metrics.embedding_dot_product),
        color: "bg-neon-purple",
        joke: dotMatrixFlavor(metrics.embedding_dot_product),
      },
      {
        label: "Vibe Check",
        value: metrics.cosine_similarity_score,
        color: "bg-neon-cyan",
        joke: vibeCheckFlavor(metrics.cosine_similarity_score),
      },
      {
        label: "Mind-Reading Score",
        value: metrics.llm_similarity_score,
        color: "bg-neon-pink",
        joke: mindReadingFlavor(metrics.llm_similarity_score, llmComputed),
      },
      {
        label: "The Verdict",
        value: metrics.final_similarity_score,
        color: "bg-neon-magenta",
        joke: verdictFlavor(metrics.final_similarity_score, metrics.considered_won),
      },
    ],
    [metrics, llmComputed],
  );

  const nearMiss = nearMissFlavor(metrics.final_similarity_score, metrics.considered_won);

  return (
    <motion.div
      initial={{ opacity: 0, y: 10, scale: 0.98 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      className="card-panel flex flex-col gap-4 border-neon-cyan/20 p-4 3xl:p-6"
    >
      <p className="text-center text-[10px] font-bold uppercase tracking-[0.2em] text-chrome-mid">
        Your Guess, Judged By Four Robots
      </p>
      {nearMiss && (
        <p className="rounded-md border border-neon-amber/40 bg-neon-amber/10 px-3 py-2 text-center text-xs text-neon-amber">
          {nearMiss}
        </p>
      )}
      <div className="flex flex-col gap-3">
        {rows.map((row) => (
          <div key={row.label} className="flex flex-col gap-1">
            <div className="flex items-center justify-between text-xs">
              <span className="font-semibold uppercase tracking-wide text-chrome-mid">{row.label}</span>
              <span className="tabular-nums text-chrome-mid">{Math.round(row.value)}</span>
            </div>
            <ProgressMeter value={row.value} colorClassName={row.color} />
            <p className="text-sm text-chrome-light">{row.joke}</p>
          </div>
        ))}
      </div>
    </motion.div>
  );
}
