"use client";

import { Toaster as Sonner } from "sonner";

export function Toaster() {
  return (
    <Sonner
      theme="dark"
      position="top-center"
      toastOptions={{
        style: {
          background: "#160a2e",
          border: "1px solid rgba(34,232,240,0.3)",
          color: "#f4f4f8",
        },
      }}
    />
  );
}
