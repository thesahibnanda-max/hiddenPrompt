import { useAuthStore } from "@/stores/auth-store";
import { ApiError, RateLimitError, UnauthorizedError } from "./errors";
import type { ApiErrorBody } from "./types";

const BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

interface ApiFetchOptions {
  method?: "GET" | "POST";
  body?: unknown;
  query?: Record<string, string>;
  /** Set false for endpoints that must NOT send X-Auth-Token (signup/login/health). Default true. */
  auth?: boolean;
}

export async function apiFetch<T>(path: string, opts: ApiFetchOptions = {}): Promise<T> {
  const { method = "GET", body, query, auth = true } = opts;

  const url = new URL(BASE_URL.replace(/\/$/, "") + path);
  if (query) {
    Object.entries(query).forEach(([k, v]) => url.searchParams.set(k, v));
  }

  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (auth) {
    const token = useAuthStore.getState().token;
    if (token) headers["X-Auth-Token"] = token;
  }

  let res: Response;
  try {
    res = await fetch(url.toString(), {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });
  } catch {
    throw new ApiError(0, {
      error_message: "Network hiccup — is the game server awake?",
      show_response_as_is: true,
    });
  }

  const timeTaken = res.headers.get("X-Time-Taken");
  if (process.env.NODE_ENV === "development" && timeTaken) {
    // eslint-disable-next-line no-console
    console.debug(`[api] ${method} ${path} — ${timeTaken}`);
  }

  if (res.status === 204 || res.status === 304) {
    return undefined as T;
  }

  const text = await res.text();
  let json: unknown = null;
  if (text) {
    try {
      json = JSON.parse(text);
    } catch {
      json = null;
    }
  }

  if (!res.ok) {
    const errBody = (json ?? { error_message: `Request failed (${res.status})` }) as ApiErrorBody;

    if (res.status === 429) {
      const retryAfterRaw = Number(res.headers.get("Retry-After") ?? "5");
      const retryAfter = Number.isFinite(retryAfterRaw) ? retryAfterRaw : 5;
      throw new RateLimitError(errBody, retryAfter);
    }
    if (res.status === 401) {
      throw new UnauthorizedError(errBody);
    }
    throw new ApiError(res.status, errBody);
  }

  // Capture the auth token off successful signup/login responses.
  const token = res.headers.get("X-Auth-Token");
  if (token && (path === "/users/signup" || path === "/users/login")) {
    useAuthStore.getState().setToken(token);
  }

  return json as T;
}
