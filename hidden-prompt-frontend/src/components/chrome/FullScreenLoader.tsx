import { Loader2 } from "lucide-react";

/**
 * Shown for the entire window where auth state is unknown or a redirect is
 * in flight — replaces every bare `return null` that used to make a real
 * hydration/auth delay look like a broken blank page. Deliberately renders
 * a solid, high-contrast panel of its own (not just a spinner floating over
 * SynthwaveBackground) and appears with zero transition — a brief blank-
 * feeling flash was traced back to this being too subtle against the busy
 * background, not to any actual stuck state.
 */
export function FullScreenLoader() {
  return (
    <div className="fixed inset-0 z-40 flex min-h-screen flex-col items-center justify-center gap-4 bg-void">
      <Loader2 className="animate-spin text-neon-cyan" size={48} strokeWidth={2.5} />
      <span className="chrome-text text-lg font-bold tracking-[0.3em]">HIDDEN PROMPT</span>
    </div>
  );
}
