"use client";

import { motion } from "framer-motion";
import type { LucideIcon } from "lucide-react";

export function StepCard({
  index,
  icon: Icon,
  title,
  body,
}: {
  index: number;
  icon: LucideIcon;
  title: string;
  body: string;
}) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 16, scale: 0.98 }}
      whileInView={{ opacity: 1, y: 0, scale: 1 }}
      viewport={{ once: true, margin: "-40px" }}
      transition={{ duration: 0.35, delay: index * 0.06 }}
      className="card-panel flex flex-col gap-3 p-5 3xl:p-8"
    >
      <div className="flex h-10 w-10 items-center justify-center rounded-full bg-neon-purple/15 text-neon-purple 3xl:h-14 3xl:w-14">
        <Icon size={20} />
      </div>
      <h3 className="font-display text-sm font-bold text-chrome-light 3xl:text-lg">{title}</h3>
      <p className="text-sm text-chrome-mid 3xl:text-base">{body}</p>
    </motion.div>
  );
}
