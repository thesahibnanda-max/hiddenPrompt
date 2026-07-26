import { pick } from "./pick";

export function genericFallback(): string {
  return pick([
    "Something broke behind the curtain. Not your fault. Probably.",
    "The robots are having a moment. Give it another shot.",
    "Technical gremlins are loose in the system.",
  ]);
}

export function rateLimitCopy(seconds: number): string {
  return `Cooldown in progress. Even robots need a nap — try again in ${seconds}s.`;
}

export function networkErrorCopy(): string {
  return "Network hiccup — is the game server awake?";
}
