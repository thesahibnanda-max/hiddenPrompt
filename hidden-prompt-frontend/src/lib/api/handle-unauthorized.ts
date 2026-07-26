import { toast } from "sonner";
import { useAuthStore } from "@/stores/auth-store";
import { UnauthorizedError } from "./errors";

/**
 * Shared handler for the one case a genuinely protected call can hit a 401:
 * the stored token has died (corrupted/rotated). Just clears the token —
 * AuthGate reactively detects the token loss on any guarded route and
 * navigates away on its own; this must not also call router.replace, or it
 * races AuthGate's own navigation the same way the old scattered guards did.
 */
export function handleUnauthorized(err: unknown): boolean {
  if (err instanceof UnauthorizedError) {
    useAuthStore.getState().logout();
    toast.error("Session expired — log back in, it happens to the best of us.");
    return true;
  }
  return false;
}
