"use client";

import { useState } from "react";
import { Music2, Pause, Play, Shuffle, SkipBack, SkipForward } from "lucide-react";
import type { Track } from "@/lib/api/types";
import { useAudioStore } from "@/stores/audio-store";
import { useMusicStore } from "@/stores/music-store";
import { howlerManager } from "@/lib/audio/howler-manager";
import { cn } from "@/lib/utils/cn";
import { SeekBar } from "./SeekBar";

interface MiniPlayerViewProps {
  tracks: Track[];
}

export function MiniPlayerView({ tracks }: MiniPlayerViewProps) {
  const playing = useAudioStore((s) => s.playing);
  const setPlaying = useAudioStore((s) => s.setPlaying);
  const currentIndex = useMusicStore((s) => s.currentIndex);
  const shuffleEnabled = useMusicStore((s) => s.shuffleEnabled);
  const toggleShuffle = useMusicStore((s) => s.toggleShuffle);
  const canGoBack = useMusicStore((s) => s.canGoBack);
  const [imgError, setImgError] = useState(false);

  const current = tracks[currentIndex];

  // Identical wiring to the old PlayPauseButton's toggle().
  const togglePlay = () => {
    if (playing) {
      setPlaying(false);
      howlerManager.pause();
      return;
    }
    setPlaying(true);
    if (howlerManager.isStarted()) {
      howlerManager.resume();
    } else {
      howlerManager.start();
    }
  };

  if (!current) {
    return <p className="p-4 text-center text-xs text-chrome-mid">No tracks yet.</p>;
  }

  return (
    <div className="flex flex-col items-center gap-3 p-4">
      <div className="flex h-32 w-32 items-center justify-center overflow-hidden rounded-lg bg-white/5 shadow-neon">
        {imgError ? (
          <Music2 size={32} className="text-chrome-mid" />
        ) : (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={current.artwork_link}
            alt=""
            width={128}
            height={128}
            className="h-full w-full object-cover"
            onError={() => setImgError(true)}
          />
        )}
      </div>
      <div className="max-w-full text-center">
        <p className="truncate text-sm font-medium text-chrome-light">{current.title}</p>
        {current.artists.length > 0 && (
          <p className="truncate text-xs text-chrome-mid">{current.artists.join(", ")}</p>
        )}
      </div>
      <SeekBar />
      <div className="flex items-center gap-4">
        <button
          type="button"
          onClick={() => howlerManager.previous()}
          disabled={!canGoBack}
          aria-label="Previous track"
          className="text-chrome-mid transition-colors hover:text-neon-cyan disabled:opacity-30 disabled:hover:text-chrome-mid"
        >
          <SkipBack size={18} />
        </button>
        <button
          type="button"
          onClick={togglePlay}
          aria-label={playing ? "Pause music" : "Play music"}
          className={cn(
            "flex h-11 w-11 items-center justify-center rounded-full border border-neon-cyan/40 bg-void-2/80 text-neon-cyan backdrop-blur-md transition-transform hover:scale-105 active:scale-95",
            playing && "animate-glow-pulse",
          )}
        >
          {playing ? <Pause size={20} /> : <Play size={20} />}
        </button>
        <button
          type="button"
          onClick={() => howlerManager.next()}
          aria-label="Next track"
          className="text-chrome-mid transition-colors hover:text-neon-cyan"
        >
          <SkipForward size={18} />
        </button>
      </div>
      <button
        type="button"
        onClick={toggleShuffle}
        aria-label={shuffleEnabled ? "Disable shuffle" : "Enable shuffle"}
        className={cn(
          "flex items-center gap-1.5 text-[10px] uppercase tracking-widest transition-colors",
          shuffleEnabled ? "text-neon-cyan" : "text-chrome-mid hover:text-neon-cyan",
        )}
      >
        <Shuffle size={14} />
        Shuffle
      </button>
    </div>
  );
}
