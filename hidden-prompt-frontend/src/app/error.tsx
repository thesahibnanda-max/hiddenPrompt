"use client";

import { useEffect } from "react";
import { Button } from "@/components/ui/button";

export default function GlobalError({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  useEffect(() => {
    // eslint-disable-next-line no-console
    console.error(error);
  }, [error]);

  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-4 px-4 text-center">
      <h1 className="chrome-text text-4xl font-black">WELP.</h1>
      <p className="max-w-sm text-chrome-mid">Something broke behind the curtain. Not your fault. Probably.</p>
      <Button onClick={reset}>Try again</Button>
    </div>
  );
}
