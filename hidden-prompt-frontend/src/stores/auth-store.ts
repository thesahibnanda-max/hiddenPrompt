import { create } from "zustand";
import { persist } from "zustand/middleware";

interface AuthState {
  token: string | null;
  email: string | null;
  // Optimistic verification flag. Not derived from any single source of
  // truth (the backend has no /users/me): starts false on fresh signup
  // (authoritative there), flips true on a successful /users/verify/decide,
  // and gets corrected back to false if a protected call (in practice,
  // POST /puzzle) ever returns the 403 "please verify" error — see
  // AuthGate/route-rules.ts. Deliberately NOT reset on logout (see logout()
  // below) — it persists per-browser so the same user logging back in
  // isn't sent through verification again; a different user sharing the
  // browser self-heals via the same 403 correction path.
  verified: boolean;
  hasHydrated: boolean;
  setToken: (token: string) => void;
  setEmail: (email: string) => void;
  setVerified: (v: boolean) => void;
  setHasHydrated: (v: boolean) => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      token: null,
      email: null,
      verified: false,
      hasHydrated: false,
      setToken: (token) => set({ token }),
      setEmail: (email) => set({ email }),
      setVerified: (verified) => set({ verified }),
      setHasHydrated: (hasHydrated) => set({ hasHydrated }),
      // Deliberately leaves `verified` untouched — see the field's comment
      // above. Only clears the actual session artifacts.
      logout: () => set({ token: null, email: null }),
    }),
    {
      name: "hidden-prompt-auth",
      onRehydrateStorage: () => (state) => {
        state?.setHasHydrated(true);
      },
    },
  ),
);
