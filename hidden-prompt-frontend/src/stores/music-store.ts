import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { Track } from "@/lib/api/types";

interface MusicState {
  tracks: Track[];
  currentIndex: number;
  panelExpanded: boolean;
  shuffleEnabled: boolean;
  // True once the first track's audio has finished loading (or failed) or
  // the catalog is empty - set by howler-manager's preloadFirstTrack, read
  // by MusicPlayerPanel to gate the loading message. Deliberately not a
  // durable preference, see partialize below.
  audioReady: boolean;
  // Mirrors howler-manager's private history stack (true iff it's
  // non-empty) purely so MiniPlayerView can dim the Previous button when
  // there's genuinely nothing before the current track, instead of the
  // button silently doing nothing.
  canGoBack: boolean;
  setTracks: (tracks: Track[]) => void;
  setCurrentIndex: (index: number) => void;
  togglePanel: () => void;
  toggleShuffle: () => void;
  setAudioReady: (ready: boolean) => void;
  setCanGoBack: (canGoBack: boolean) => void;
}

export const useMusicStore = create<MusicState>()(
  persist(
    (set) => ({
      // Refetched on every page load (see useTracks) - deliberately excluded
      // from persistence below via partialize, so a stale catalog from a
      // prior visit never lingers in localStorage.
      tracks: [],
      currentIndex: 0,
      // Starts expanded so the full catalog (artwork + genre) is visible the
      // moment the page loads - except on narrow/mobile viewports, where
      // starting collapsed to the mini-player avoids covering the guess
      // input or other controls on a short screen. Only affects the initial
      // in-memory value before persist rehydrates a stored choice; the
      // arrow toggle then remembers whichever state the user leaves it in,
      // across reloads, on any device.
      panelExpanded: typeof window !== "undefined" ? window.innerWidth >= 640 : true,
      shuffleEnabled: true,
      // Re-evaluated fresh every load (tied to a fresh preload each time),
      // so this is excluded from persistence just like tracks/currentIndex.
      audioReady: false,
      canGoBack: false,
      setTracks: (tracks) => set({ tracks }),
      setCurrentIndex: (currentIndex) => set({ currentIndex }),
      togglePanel: () => set((s) => ({ panelExpanded: !s.panelExpanded })),
      toggleShuffle: () => set((s) => ({ shuffleEnabled: !s.shuffleEnabled })),
      setAudioReady: (audioReady) => set({ audioReady }),
      setCanGoBack: (canGoBack) => set({ canGoBack }),
    }),
    {
      name: "hidden-prompt-music",
      partialize: (s) => ({ panelExpanded: s.panelExpanded, shuffleEnabled: s.shuffleEnabled }),
    },
  ),
);
