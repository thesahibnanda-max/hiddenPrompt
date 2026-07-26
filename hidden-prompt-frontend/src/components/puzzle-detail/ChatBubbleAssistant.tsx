"use client";

import { motion } from "framer-motion";
import { Bot } from "lucide-react";
import { formatTimestamp } from "@/lib/utils/format";
import type { MessageElement } from "@/lib/api/types";

export function ChatBubbleAssistant({ message }: { message: MessageElement }) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0 }}
      layout
      className="flex flex-col items-start gap-1"
    >
      <div className="flex items-center gap-1.5 pl-1 text-[10px] uppercase tracking-widest text-neon-cyan/80">
        <Bot size={12} />
        Hint
      </div>
      <div className="max-w-[85%] break-words rounded-lg rounded-tl-sm border border-neon-cyan/40 bg-neon-cyan/10 px-4 py-2.5 text-sm text-chrome-light sm:max-w-2xl 3xl:max-w-4xl 3xl:text-base">
        {message.message}
      </div>
      <span className="pl-1 text-[10px] text-chrome-mid/70">{formatTimestamp(message.timestamp)}</span>
    </motion.div>
  );
}
