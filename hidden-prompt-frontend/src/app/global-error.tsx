"use client";

import { useEffect } from "react";

// Root-level safety net: error.tsx only catches exceptions thrown by page
// content, not by RootLayout's own tree (AuthGate, SynthwaveBackground,
// AudioBootstrap). Without this file, an exception thrown there produces a
// genuinely blank white screen with zero recovery UI - this renders its own
// <html>/<body> (required for global-error.tsx specifically) since it
// replaces the root layout entirely when triggered.
export default function GlobalError({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  useEffect(() => {
    // eslint-disable-next-line no-console
    console.error(error);
  }, [error]);

  return (
    <html>
      <body style={{ background: "#0a0118", color: "#e8e6f0" }}>
        <div
          style={{
            display: "flex",
            minHeight: "100vh",
            flexDirection: "column",
            alignItems: "center",
            justifyContent: "center",
            gap: "1rem",
            padding: "1rem",
            textAlign: "center",
            fontFamily: "system-ui, sans-serif",
          }}
        >
          <h1 style={{ fontSize: "2.25rem", fontWeight: 900 }}>WELP.</h1>
          <p style={{ maxWidth: "24rem", opacity: 0.8 }}>
            Something broke behind the curtain. Not your fault. Probably.
          </p>
          <button
            onClick={reset}
            style={{
              padding: "0.5rem 1.25rem",
              borderRadius: "0.5rem",
              border: "1px solid #22e8f0",
              color: "#22e8f0",
              background: "transparent",
              cursor: "pointer",
            }}
          >
            Try again
          </button>
        </div>
      </body>
    </html>
  );
}
