"use client";

import { useEffect } from "react";
import { useAudioStore } from "@/stores/audio-store";
import { howlerManager } from "@/lib/audio/howler-manager";

/**
 * Unlocks playback on the first user gesture anywhere on the page, honoring
 * a previously-playing preference without needing that gesture to be the
 * play/pause button specifically. Deliberately does NOT construct any Howl
 * instances until that gesture fires - Howler preloads its audio source on
 * construction regardless of whether playback actually starts, so an eager
 * init on every route mount was quietly downloading ~2.9MB of audio on
 * every single page view, including routes nobody ever plays a puzzle on.
 * A pre-gesture "try to start immediately" attempt was removed for the same
 * reason - browsers block it unconditionally anyway (autoplay policy, not a
 * bug), so it never succeeded and only paid the preload cost for nothing.
 */
export function AudioBootstrap() {
  const playing = useAudioStore((s) => s.playing);
  const hasHydrated = useAudioStore((s) => s.hasHydrated);

  // Arms Howler's own gesture-based unlock mechanism immediately on
  // mount, independent of hasHydrated/playing below - it's what actually
  // makes the very first click/keypress anywhere unlock audio, so it
  // needs to be armed before that gesture can happen, not gated behind
  // the same checks that decide whether to *start playback*.
  useEffect(() => {
    howlerManager.primeUnlock();
  }, []);

  useEffect(() => {
    if (!hasHydrated) return;
    if (!playing) return;

    const unlock = () => {
      howlerManager.start();
      window.removeEventListener("pointerdown", unlock);
      window.removeEventListener("keydown", unlock);
    };
    window.addEventListener("pointerdown", unlock, { once: true });
    window.addEventListener("keydown", unlock, { once: true });
    return () => {
      window.removeEventListener("pointerdown", unlock);
      window.removeEventListener("keydown", unlock);
    };
  }, [hasHydrated, playing]);

  return null;
}
