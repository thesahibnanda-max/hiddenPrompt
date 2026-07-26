"use client";

import { useEffect, useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { Loader2 } from "lucide-react";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import { PUZZLE_LOADING_LINES } from "@/lib/copy/loading-flavor";

export function NewPuzzleLoadingOverlay({ open }: { open: boolean }) {
  const [lineIndex, setLineIndex] = useState(0);

  useEffect(() => {
    if (!open) return;
    const id = setInterval(() => {
      setLineIndex((i) => (i + 1) % PUZZLE_LOADING_LINES.length);
    }, 2500);
    return () => clearInterval(id);
  }, [open]);

  return (
    <Dialog open={open}>
      <DialogContent
        className="flex flex-col items-center gap-5 text-center [&>button]:hidden"
        onInteractOutside={(e) => e.preventDefault()}
        onEscapeKeyDown={(e) => e.preventDefault()}
      >
        <DialogTitle className="sr-only">Generating your puzzle</DialogTitle>
        <Loader2 className="animate-spin text-neon-cyan" size={40} />
        <AnimatePresence mode="wait">
          <motion.p
            key={lineIndex}
            initial={{ opacity: 0, y: 6 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -6 }}
            className="text-sm text-chrome-mid"
          >
            {PUZZLE_LOADING_LINES[lineIndex]}
          </motion.p>
        </AnimatePresence>
      </DialogContent>
    </Dialog>
  );
}
