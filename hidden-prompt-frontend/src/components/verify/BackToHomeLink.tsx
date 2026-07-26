"use client";

import { useAuthStore } from "@/stores/auth-store";

/**
 * For a user who typed the wrong email at signup and only realizes it here.
 * Just clears the token — AuthGate reactively detects the loss and routes
 * back to "/" on its own, landing on a fresh, empty signup form.
 */
export function BackToHomeLink() {
  const logout = useAuthStore((s) => s.logout);
  return (
    <button
      type="button"
      onClick={logout}
      className="text-xs text-chrome-mid underline-offset-4 hover:text-neon-magenta hover:underline"
    >
      Wrong email? Log out and start over
    </button>
  );
}
