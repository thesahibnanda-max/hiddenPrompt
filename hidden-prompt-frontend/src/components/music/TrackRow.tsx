"use client";

import { useState } from "react";
import { Music2 } from "lucide-react";
import type { Track } from "@/lib/api/types";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils/cn";

interface TrackRowProps {
  track: Track;
  active: boolean;
  onClick: () => void;
}

export function TrackRow({ track, active, onClick }: TrackRowProps) {
  const [imgError, setImgError] = useState(false);

  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "flex w-full items-center gap-3 border-b border-white/5 px-3 py-2 text-left transition-colors hover:bg-white/5",
        active && "bg-neon-cyan/10",
      )}
    >
      <div className="flex h-10 w-10 shrink-0 items-center justify-center overflow-hidden rounded-md bg-white/5">
        {imgError ? (
          <Music2 size={16} className="text-chrome-mid" />
        ) : (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={track.artwork_link}
            alt=""
            width={40}
            height={40}
            className="h-full w-full object-cover"
            loading="lazy"
            onError={() => setImgError(true)}
          />
        )}
      </div>
      <div className="min-w-0 flex-1">
        <p className={cn("truncate text-sm font-medium", active ? "text-neon-cyan" : "text-chrome-light")}>
          {track.title}
        </p>
        {track.artists.length > 0 && <p className="truncate text-xs text-chrome-mid">{track.artists.join(", ")}</p>}
        {track.genre.length > 0 && (
          <div className="mt-1 flex flex-wrap gap-1">
            {track.genre.map((g) => (
              <Badge key={g} variant="neutral" className="px-1.5 py-0 text-[9px]">
                {g}
              </Badge>
            ))}
          </div>
        )}
      </div>
    </button>
  );
}
