"use client";

import type { Track } from "@/lib/api/types";
import { howlerManager } from "@/lib/audio/howler-manager";
import { useMusicStore } from "@/stores/music-store";
import { TrackRow } from "./TrackRow";

interface TrackListViewProps {
  tracks: Track[];
}

export function TrackListView({ tracks }: TrackListViewProps) {
  const currentIndex = useMusicStore((s) => s.currentIndex);

  if (tracks.length === 0) {
    return <p className="p-4 text-center text-xs text-chrome-mid">No tracks yet.</p>;
  }

  return (
    <div className="max-h-80 overflow-y-auto">
      {tracks.map((track, i) => (
        <TrackRow
          key={track.id}
          track={track}
          active={i === currentIndex}
          onClick={() => howlerManager.playTrackById(track.id)}
        />
      ))}
    </div>
  );
}
