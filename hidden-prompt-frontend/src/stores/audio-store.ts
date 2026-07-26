import { create } from "zustand";
import { persist } from "zustand/middleware";

interface AudioState {
  playing: boolean;
  hasHydrated: boolean;
  setPlaying: (playing: boolean) => void;
  setHasHydrated: (v: boolean) => void;
}

export const useAudioStore = create<AudioState>()(
  persist(
    (set) => ({
      // App's own preference is sound on. True zero-gesture autoplay is
      // blocked by every modern browser regardless of this value, so
      // AudioBootstrap attempts to start immediately and falls back to
      // starting on the very first click/keypress anywhere on the page —
      // this is the most "on by default" any site can honestly be.
      playing: true,
      hasHydrated: false,
      setPlaying: (playing) => set({ playing }),
      setHasHydrated: (hasHydrated) => set({ hasHydrated }),
    }),
    {
      name: "hidden-prompt-audio",
      onRehydrateStorage: () => (state) => {
        state?.setHasHydrated(true);
      },
    },
  ),
);
