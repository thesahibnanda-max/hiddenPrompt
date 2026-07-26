"use client";

import { useCallback, useEffect, useState } from "react";
import { getTracks } from "@/lib/api/endpoints";
import { howlerManager } from "@/lib/audio/howler-manager";
import { useMusicStore } from "@/stores/music-store";

export function useTracks() {
  const tracks = useMusicStore((s) => s.tracks);
  const setTracks = useMusicStore((s) => s.setTracks);
  const [loading, setLoading] = useState(tracks.length === 0);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    if (tracks.length === 0) setLoading(true);
    try {
      const data = await getTracks();
      setTracks(data);
      howlerManager.setTracks(data);
      setError(null);
    } catch {
      setError("Couldn't load the music library.");
    } finally {
      setLoading(false);
    }
  }, [setTracks, tracks.length]);

  useEffect(() => {
    refresh();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return { tracks, loading, error, refresh };
}
