"use client";

import { motion } from "framer-motion";
import { usePathname } from "next/navigation";

export function PageTransition({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  // No AnimatePresence: it gates mounting the new page on the old page's
  // exit animation fully resolving, which browser back/forward navigation
  // (Next's cache-reusing, startTransition-wrapped popstate handling) can
  // leave permanently unresolved - the page gets stuck rendering nothing
  // but shared chrome until a manual reload. A bare keyed motion.div still
  // gets a clean fade-in per route; the old page just unmounts immediately
  // via normal React reconciliation, with no artificial overlap window.
  return (
    <motion.div key={pathname} initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ duration: 0.15 }}>
      {children}
    </motion.div>
  );
}
