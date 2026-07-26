"use client";

import { useState } from "react";
import { AnimatePresence, motion } from "framer-motion";
import { ChevronDown, Music2, X } from "lucide-react";
import { useTracks } from "@/hooks/useTracks";
import { useMusicStore } from "@/stores/music-store";
import { cn } from "@/lib/utils/cn";
import { TrackListView } from "./TrackListView";
import { MiniPlayerView } from "./MiniPlayerView";
import { MusicLoadingState } from "./MusicLoadingState";

export function MusicPlayerPanel() {
  const { tracks, loading, error, refresh } = useTracks();
  const panelExpanded = useMusicStore((s) => s.panelExpanded);
  const togglePanel = useMusicStore((s) => s.togglePanel);
  const audioReady = useMusicStore((s) => s.audioReady);
  // Mobile-only: whether the full panel is open at all, vs. just the small
  // launcher icon. Deliberately local, un-persisted state (not the music
  // store) - always starts closed on a fresh load so nothing on a short
  // page can ever be covered at rest. Irrelevant at sm: and up, where the
  // launcher is always hidden and the full panel is always shown via CSS.
  const [mobileOpen, setMobileOpen] = useState(false);

  // "Ready" means the audio itself, not just the catalog JSON - see
  // howler-manager's preloadFirstTrack for what sets audioReady.
  const showLoading = !error && (loading || !audioReady);

  return (
    <>
      <button
        type="button"
        onClick={() => setMobileOpen(true)}
        aria-label="Open music panel"
        className={cn(
          "fixed bottom-4 right-4 z-40 flex h-12 w-12 items-center justify-center rounded-full border border-neon-cyan/40 bg-void-2/80 text-neon-cyan shadow-neon backdrop-blur-md transition-transform hover:scale-105 sm:hidden",
          mobileOpen && "hidden",
        )}
      >
        <Music2 size={20} />
      </button>
      <div
        className={cn(
          "card-panel fixed inset-x-4 bottom-4 z-40 flex-col overflow-hidden shadow-neon sm:inset-x-auto sm:right-4 sm:flex sm:w-72 3xl:w-80",
          mobileOpen ? "flex" : "hidden sm:flex",
        )}
      >
        <div className="flex items-center justify-between border-b border-white/10 px-3 py-2">
          <span className="text-[10px] uppercase tracking-widest text-chrome-mid">Music</span>
          <div className="flex items-center gap-1">
            <button
              type="button"
              onClick={togglePanel}
              aria-label={panelExpanded ? "Collapse music panel" : "Expand music panel"}
              className="flex h-6 w-6 items-center justify-center rounded-full text-neon-cyan transition-transform hover:scale-110"
            >
              <ChevronDown
                size={16}
                className={cn("transition-transform duration-200", panelExpanded && "rotate-180")}
              />
            </button>
            <button
              type="button"
              onClick={() => setMobileOpen(false)}
              aria-label="Close music panel"
              className="flex h-6 w-6 items-center justify-center rounded-full text-neon-cyan transition-transform hover:scale-110 sm:hidden"
            >
              <X size={16} />
            </button>
          </div>
        </div>
        <AnimatePresence mode="wait" initial={false}>
        {error ? (
          <motion.div
            key="error"
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: 0.25, ease: "easeInOut" }}
          >
            <div className="flex flex-col items-center gap-2 p-4 text-center">
              <p className="text-xs text-chrome-mid">{error}</p>
              <button
                type="button"
                onClick={refresh}
                className="text-xs font-semibold text-neon-cyan underline underline-offset-2"
              >
                Retry
              </button>
            </div>
          </motion.div>
        ) : showLoading ? (
          <motion.div
            key="loading"
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: 0.25, ease: "easeInOut" }}
          >
            <MusicLoadingState />
          </motion.div>
        ) : panelExpanded ? (
          <motion.div
            key="list"
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: 0.25, ease: "easeInOut" }}
          >
            <TrackListView tracks={tracks} />
          </motion.div>
        ) : (
          <motion.div
            key="mini"
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: 0.25, ease: "easeInOut" }}
          >
            <MiniPlayerView tracks={tracks} />
          </motion.div>
        )}
        </AnimatePresence>
      </div>
    </>
  );
}
