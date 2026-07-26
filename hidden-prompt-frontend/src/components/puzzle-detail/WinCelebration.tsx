"use client";

import { useMemo } from "react";
import { motion } from "framer-motion";

import { Trophy } from "lucide-react";
import { winHeadline, winSubline } from "@/lib/copy/win-flavor";
import { formatScore } from "@/lib/utils/format";

export function WinCelebration({
  finalScore,
  isFloatPrecision,
}: {
  finalScore: number;
  isFloatPrecision: boolean;
}) {
  const headline = useMemo(() => winHeadline(), []);
  const score = formatScore(finalScore, isFloatPrecision);
  const subline = useMemo(() => winSubline(score), [score]);

  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.9 }}
      animate={{ opacity: 1, scale: 1 }}
      transition={{ type: "spring", stiffness: 200, damping: 16 }}
      className="card-panel flex flex-col items-center gap-3 border-neon-cyan/40 p-8 text-center shadow-neon"
    >
      <Trophy className="animate-glow-pulse text-neon-cyan" size={40} />
      <h2 className="chrome-text text-2xl font-black tracking-wide 3xl:text-4xl">{headline}</h2>
      <p className="text-sm text-chrome-mid 3xl:text-base">{subline}</p>
    </motion.div>
  );
}
