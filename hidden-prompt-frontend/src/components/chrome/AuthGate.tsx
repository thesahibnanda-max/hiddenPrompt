"use client";

import { useEffect, useRef } from "react";
import { usePathname, useRouter } from "next/navigation";
import { useAuthStore } from "@/stores/auth-store";
import { categorizeRoute, isHardBlocked, arrivalRedirectFor, hardBlockDestination } from "@/lib/auth/route-rules";
import { FullScreenLoader } from "./FullScreenLoader";

/**
 * The single authority for "what should render right now" and "should we
 * navigate." Every previous per-page guard (useAuthGuard, useVerifiedGuard,
 * (auth)/layout.tsx, verify/page.tsx's own effect) independently owned a
 * router.replace() call and a `return null` render-guard, all reacting
 * uncoordinated to the same token/verified state — that's what produced
 * flashing/flickering/blank-page navigation. This component replaces all
 * of them. Children only ever mount once this gate has confirmed the
 * current route is authorized, so pages never need their own auth checks.
 */
export function AuthGate({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const token = useAuthStore((s) => s.token);
  const verified = useAuthStore((s) => s.verified);
  const hasHydrated = useAuthStore((s) => s.hasHydrated);

  const category = categorizeRoute(pathname);
  const hardBlocked = hasHydrated && isHardBlocked(category, token, verified);

  // Tracks the last pathname we've already let real content render for, so
  // a later in-visit state change on the SAME pathname (verified flipping
  // true while VerifyGate is mid-celebration) doesn't retroactively fire
  // the arrival-only rule. Resets naturally whenever pathname changes.
  // Only relevant to the "auth-only" (/verify) arrival rule — see
  // route-rules.ts for why "public-only" (/) is a hard-block instead.
  const clearedPathnameRef = useRef<string | null>(null);
  const isFreshArrival = clearedPathnameRef.current !== pathname;
  const arrivalRedirect =
    hasHydrated && isFreshArrival && !hardBlocked ? arrivalRedirectFor(category, verified) : null;

  useEffect(() => {
    if (!hasHydrated) return;

    if (hardBlocked) {
      router.replace(hardBlockDestination(category, token, verified));
      return;
    }
    if (arrivalRedirect) {
      router.replace(arrivalRedirect);
      return;
    }
    clearedPathnameRef.current = pathname;
  }, [hasHydrated, hardBlocked, arrivalRedirect, pathname, category, token, verified, router]);

  if (!hasHydrated || hardBlocked || arrivalRedirect) {
    return <FullScreenLoader />;
  }

  return <>{children}</>;
}
