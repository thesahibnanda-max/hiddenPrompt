import type { ApiErrorBody } from "./types";

export class ApiError extends Error {
  readonly statusCode: number;
  readonly showAsIs: boolean;
  readonly body: ApiErrorBody;

  constructor(statusCode: number, body: ApiErrorBody) {
    super(body.error_message || "Something went wrong");
    this.name = "ApiError";
    this.statusCode = statusCode;
    // Default to false (don't blindly display) when the field is absent,
    // e.g. router-level 404/405 fallback bodies, or a malformed body.
    this.showAsIs = body.show_response_as_is === true;
    this.body = body;
  }
}

export class RateLimitError extends ApiError {
  readonly retryAfterSeconds: number;

  constructor(body: ApiErrorBody, retryAfterSeconds: number) {
    super(429, body);
    this.name = "RateLimitError";
    this.retryAfterSeconds = retryAfterSeconds;
  }
}

export class UnauthorizedError extends ApiError {
  constructor(body: ApiErrorBody) {
    super(401, body);
    this.name = "UnauthorizedError";
  }
}
