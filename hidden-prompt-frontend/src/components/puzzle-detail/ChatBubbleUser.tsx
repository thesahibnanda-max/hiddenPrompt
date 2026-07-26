"use client";

import { motion } from "framer-motion";
import { formatTimestamp } from "@/lib/utils/format";
import type { MessageElement } from "@/lib/api/types";

export function ChatBubbleUser({ message }: { message: MessageElement }) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0 }}
      layout
      className="flex flex-col items-end gap-1"
    >
      <div className="max-w-[85%] break-words rounded-lg rounded-tr-sm border border-neon-magenta/40 bg-neon-magenta/10 px-4 py-2.5 text-sm text-chrome-light sm:max-w-2xl 3xl:max-w-4xl 3xl:text-base">
        {message.message}
      </div>
      <span className="pr-1 text-[10px] text-chrome-mid/70">{formatTimestamp(message.timestamp)}</span>
    </motion.div>
  );
}
