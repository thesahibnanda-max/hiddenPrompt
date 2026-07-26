"use client";

import { useEffect, useState } from "react";

/** Generic mm:ss-capable countdown. Pass seconds; returns remaining seconds, ticking down to 0. */
export function useCountdown(initialSeconds: number) {
  const [seconds, setSeconds] = useState(initialSeconds);

  useEffect(() => {
    setSeconds(initialSeconds);
  }, [initialSeconds]);

  useEffect(() => {
    if (seconds <= 0) return;
    const id = setInterval(() => {
      setSeconds((s) => Math.max(0, s - 1));
    }, 1000);
    return () => clearInterval(id);
  }, [seconds > 0]); // eslint-disable-line react-hooks/exhaustive-deps

  const formatted = `${Math.floor(seconds / 60)
    .toString()
    .padStart(1, "0")}:${(seconds % 60).toString().padStart(2, "0")}`;

  return { seconds, formatted, done: seconds <= 0 };
}
