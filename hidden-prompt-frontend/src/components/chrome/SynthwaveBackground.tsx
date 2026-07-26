export function SynthwaveBackground() {
  return (
    <div aria-hidden="true" className="pointer-events-none fixed inset-0 -z-10 overflow-hidden bg-void">
      {/* radial "sun" glow - kept off the GPU-heaviest blur tier and given a
          compositing hint, since this sits fixed + blurred behind every
          route with no way to ever go off-screen */}
      <div
        className="absolute left-1/2 top-1/3 h-[45vmax] w-[45vmax] -translate-x-1/2 -translate-y-1/2 rounded-full opacity-30 blur-2xl [will-change:transform]"
        style={{
          background:
            "radial-gradient(circle, rgba(255,179,71,0.55) 0%, rgba(255,110,199,0.35) 35%, rgba(157,77,255,0.15) 60%, transparent 75%)",
        }}
      />
      {/* scrolling neon grid horizon */}
      <div className="absolute inset-x-0 bottom-0 h-2/3 animate-grid-scroll bg-grid-horizon bg-[length:100%_40px] [will-change:background-position]" />
      {/* scanlines */}
      <div className="scanline-overlay" />
    </div>
  );
}
