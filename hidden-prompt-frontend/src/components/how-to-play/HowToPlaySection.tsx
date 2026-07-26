"use client";

import { MessageSquareQuote, Brain, Gauge, Trophy } from "lucide-react";
import { HOW_TO_PLAY_STEPS } from "@/lib/copy/how-to-play-copy";
import { StepCard } from "./StepCard";

const ICONS = [MessageSquareQuote, Brain, Gauge, Trophy];

export function HowToPlaySection() {
  return (
    <section className="w-full max-w-4xl">
      <h2 className="chrome-text mb-6 text-center text-2xl font-black tracking-wide 3xl:text-4xl">
        HOW TO PLAY
      </h2>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {HOW_TO_PLAY_STEPS.map((step, i) => (
          <StepCard key={step.title} index={i} icon={ICONS[i]} title={step.title} body={step.body} />
        ))}
      </div>
    </section>
  );
}
