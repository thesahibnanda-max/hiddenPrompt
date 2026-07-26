"use client";

import { useEffect, useRef, useState } from "react";
import { useAudioStore } from "@/stores/audio-store";
import { useMusicStore } from "@/stores/music-store";
import { howlerManager } from "@/lib/audio/howler-manager";
import { cn } from "@/lib/utils/cn";

function formatTime(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 0) return "0:00";
  const mins = Math.floor(seconds / 60);
  const secs = Math.floor(seconds % 60);
  return `${mins}:${secs.toString().padStart(2, "0")}`;
}

export function SeekBar() {
  const playing = useAudioStore((s) => s.playing);
  const currentIndex = useMusicStore((s) => s.currentIndex);
  const [position, setPosition] = useState(0);
  const [duration, setDuration] = useState(0);
  const isDragging = useRef(false);

  // A new track was selected - stale numbers from the previous song
  // shouldn't linger, and duration may not be known yet if this track's
  // Howl was just lazily built.
  useEffect(() => {
    setPosition(howlerManager.seek());
    setDuration(howlerManager.getDuration());
  }, [currentIndex]);

  // Howler has no continuous "timeupdate" event, so polling is the
  // standard pattern for a progress bar with it - ~250ms is smooth enough
  // without a raw requestAnimationFrame loop's complexity. Skipped while
  // paused (nothing moving to poll) and while the user is actively
  // dragging the thumb, so the interval never fights the drag.
  useEffect(() => {
    if (!playing) return;
    const id = setInterval(() => {
      if (isDragging.current) return;
      setPosition(howlerManager.seek());
      setDuration(howlerManager.getDuration());
    }, 250);
    return () => clearInterval(id);
  }, [playing, currentIndex]);

  const ready = duration > 0;
  const percent = ready ? (position / duration) * 100 : 0;

  return (
    <div className="flex w-full items-center gap-2">
      <span className="w-8 shrink-0 text-right text-[10px] text-chrome-mid">{formatTime(position)}</span>
      <input
        type="range"
        min={0}
        max={ready ? duration : 1}
        step={0.1}
        value={ready ? position : 0}
        disabled={!ready}
        onPointerDown={() => {
          isDragging.current = true;
        }}
        onPointerUp={() => {
          isDragging.current = false;
        }}
        onChange={(e) => {
          const value = Number(e.target.value);
          setPosition(value);
          howlerManager.seek(value);
        }}
        aria-label="Seek"
        className={cn(
          "h-1 w-full flex-1 cursor-pointer appearance-none rounded-full disabled:cursor-default",
          "[&::-webkit-slider-thumb]:h-3 [&::-webkit-slider-thumb]:w-3 [&::-webkit-slider-thumb]:appearance-none [&::-webkit-slider-thumb]:rounded-full [&::-webkit-slider-thumb]:bg-neon-cyan [&::-webkit-slider-thumb]:shadow-neon",
          "[&::-moz-range-thumb]:h-3 [&::-moz-range-thumb]:w-3 [&::-moz-range-thumb]:rounded-full [&::-moz-range-thumb]:border-0 [&::-moz-range-thumb]:bg-neon-cyan",
        )}
        style={{
          background: `linear-gradient(to right, #22e8f0 ${percent}%, rgba(255,255,255,0.1) ${percent}%)`,
        }}
      />
      <span className="w-8 shrink-0 text-[10px] text-chrome-mid">{formatTime(duration)}</span>
    </div>
  );
}
