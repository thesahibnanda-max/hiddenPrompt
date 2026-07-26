"use client";

import dynamic from "next/dynamic";
import { Music2 } from "lucide-react";
import { MusicLoadingState } from "@/components/music/MusicLoadingState";

// Both dynamically imported (not just MusicPlayerPanel) so `howler`'s
// bundle weight genuinely defers past first paint on every route,
// including ones that never touch music - AudioBootstrap also statically
// imports howler-manager (and therefore howler) on its own, so leaving it
// as a normal import here would still pull the same dependency in
// eagerly regardless of what MusicPlayerPanel does. `ssr: false` requires
// a Client Component boundary, which is what this wrapper exists to
// provide - the root layout itself stays a Server Component.
const AudioBootstrap = dynamic(() => import("./AudioBootstrap").then((m) => m.AudioBootstrap), { ssr: false });

// Mirrors MusicPlayerPanel's real shell exactly (same position/width
// classes, and the same mobile-icon-vs-desktop-panel split) so there's no
// layout shift when the real chunk swaps in - this is what keeps the panel
// appearing with the rest of the page instead of popping in once the
// chunk finishes loading. No chevron/close buttons here, since there's
// nothing to toggle until the real component mounts.
function MusicPanelSkeleton() {
  return (
    <>
      <div className="fixed bottom-4 right-4 z-40 flex h-12 w-12 animate-pulse items-center justify-center rounded-full border border-neon-cyan/40 bg-void-2/80 text-neon-cyan shadow-neon backdrop-blur-md sm:hidden">
        <Music2 size={20} />
      </div>
      <div className="card-panel fixed inset-x-4 bottom-4 z-40 hidden overflow-hidden shadow-neon sm:inset-x-auto sm:right-4 sm:block sm:w-72 3xl:w-80">
        <div className="flex items-center justify-between border-b border-white/10 px-3 py-2">
          <span className="text-[10px] uppercase tracking-widest text-chrome-mid">Music</span>
        </div>
        <MusicLoadingState />
      </div>
    </>
  );
}

const MusicPlayerPanel = dynamic(() => import("@/components/music/MusicPlayerPanel").then((m) => m.MusicPlayerPanel), {
  ssr: false,
  loading: () => <MusicPanelSkeleton />,
});

export function MusicSystem() {
  return (
    <>
      <AudioBootstrap />
      <MusicPlayerPanel />
    </>
  );
}
