"use client";

import { motion } from "framer-motion";
import { Eye } from "lucide-react";

export function HiddenPromptReveal({ hiddenPrompt }: { hiddenPrompt: string }) {
  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.95 }}
      animate={{ opacity: 1, scale: 1 }}
      transition={{ delay: 0.2 }}
      className="card-panel flex flex-col items-center gap-2 border-neon-purple/40 p-6 text-center"
    >
      <div className="flex items-center gap-1.5 text-[10px] uppercase tracking-widest text-neon-purple">
        <Eye size={12} />
        The hidden prompt was
      </div>
      <p className="chrome-text break-words text-lg font-bold 3xl:text-2xl">&ldquo;{hiddenPrompt}&rdquo;</p>
    </motion.div>
  );
}
