// Pure, React-free route classification and redirect rules. The single
// source of truth for "what auth state does this route require" — see
// AuthGate, which is the only component allowed to act on these.

export type RouteCategory = "public-only" | "auth-only" | "auth-and-verified" | "unguarded";

export function categorizeRoute(pathname: string): RouteCategory {
  if (pathname === "/") return "public-only";
  if (pathname === "/verify") return "auth-only";
  if (pathname === "/dashboard" || pathname.startsWith("/puzzle/")) return "auth-and-verified";
  return "unguarded";
}

/**
 * Always-urgent: state that must kick the user out of the current route
 * immediately, with no exceptions. Includes an authed visitor landing on
 * the public-only landing page — unlike /verify's "already verified" case
 * below, there is no page-owned transition on "/" worth protecting from a
 * preemptive redirect, so this fires reactively every time, not just once
 * per arrival (that distinction is exactly what the earlier version of
 * this rule got wrong: suppressing it here meant a token appearing via
 * login/signup on an already-"cleared" "/" pathname never redirected).
 */
export function isHardBlocked(category: RouteCategory, token: string | null, verified: boolean): boolean {
  if (category === "unguarded") return false;
  if (category === "public-only") return !!token;
  if (!token) return true;
  if (category === "auth-and-verified" && !verified) return true;
  return false;
}

/**
 * "You're already past this stage" — only /verify has a legitimate case
 * here (an already-verified visitor landing on it). Evaluated ONLY once
 * per fresh arrival at a pathname (see AuthGate), never reactively
 * mid-visit, so it never preempts VerifyGate's own celebration-then-
 * redirect after a fresh OTP success, where `verified` flips true while
 * the user is still meant to see the celebration for a moment.
 */
export function arrivalRedirectFor(category: RouteCategory, verified: boolean): string | null {
  if (category === "auth-only" && verified) return "/dashboard";
  return null;
}

/** Destination for a hard-blocked route — where isHardBlocked(...) === true should redirect to. */
export function hardBlockDestination(category: RouteCategory, token: string | null, verified: boolean): string {
  if (category === "public-only") return verified ? "/dashboard" : "/verify";
  if (category === "auth-and-verified" && token) return "/verify";
  return "/";
}
