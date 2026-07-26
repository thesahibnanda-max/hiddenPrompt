# Hidden Prompt API Reference

Base URL used throughout this document: `http://localhost:8080` (the
server listens on `$PORT`, defaulting to `8080` — see
[README.md](./README.md#running-the-server)).

## Conventions

### Content type

Every request with a body must send `Content-Type: application/json`.
Every response is `application/json`.

### Authentication

Some endpoints require an `X-Auth-Token` header. You get this token from
a successful `POST /users/signup` or `POST /users/login` call — it comes
back as a **response header** (not in the JSON body), and it's your
encrypted email address. Send it back verbatim on every subsequent
authenticated request:

```
X-Auth-Token: <value copied from the signup/login response header>
```

If the header is missing, empty, or fails to decrypt, every protected
endpoint responds identically:

```json
{
  "show_response_as_is": true,
  "error_message": "Please LogIn or SignUp",
  "status_code": 401
}
```

The token never expires server-side and carries no session state — it's
purely an encrypted identity claim, so treat it like a bearer credential
(don't log it, don't put it in a URL).

A separate, smaller set of endpoints — the `/admin/*` group and
`POST /music` — require an `X-Admin-Key` header instead, checked against a
single static shared secret rather than a per-user token:

```
X-Admin-Key: <the configured admin key>
```

A missing or incorrect key gets a `401` with `error_message: "Invalid or
missing admin key."`. This is unrelated to `X-Auth-Token` — an admin-key
endpoint doesn't need a user to be logged in at all.

### Every response header

Every request, success or failure, gets an `X-Time-Taken` header (e.g.
`X-Time-Taken: 14.7712ms`) — server-side handling time for that request.

### Error response shape

Every non-2xx response body looks like this:

```json
{
  "show_response_as_is": true,
  "error_message": "The email or password you entered is incorrect.",
  "status_code": 401
}
```

| Field | Type | Meaning |
|---|---|---|
| `show_response_as_is` | bool | `true`: `error_message` is a polished, safe sentence you can display to the end user as-is. `false`: `error_message` is a lower-level/internal string (a JSON-parse failure, an unexpected `5xx`) — log it, don't show it to a user verbatim. |
| `error_message` | string | Human-readable description of what went wrong. |
| `status_code` | int | Same value as the HTTP status code, repeated in the body for convenience. |

### Rate limiting

Every endpoint below is individually rate-limited via Redis (a
GCRA/token-bucket limiter — it smooths short bursts rather than enforcing
a hard cliff at a fixed-window boundary). When you exceed a limit:

- **Status**: `429 Too Many Requests`
- **Body**: the standard error shape, `error_message: "Too many requests. Please slow down and try again shortly."`
- **Header**: `Retry-After: <seconds>` — wait at least this long before retrying.

Pre-auth endpoints (signup/login) are limited **per client IP**;
post-auth endpoints are limited **per authenticated user** (by email).
Exact limits are listed per-endpoint below.

---

## Endpoints

| Method | Path | Auth | Rate limit |
|---|---|---|---|
| `GET` | `/health` | none | none (see [README](./README.md#observability)) |
| `POST` | `/users/signup` | none | 8/hour per IP |
| `POST` | `/users/login` | none | 10/minute per IP |
| `POST` | `/users/verify/initiate` | required | 3/hour per user |
| `POST` | `/users/verify/decide` | required | 6/minute per user |
| `POST` | `/puzzle` | required | 8/hour per user |
| `GET` | `/puzzle` | required | 20/minute per user |
| `GET` | `/puzzle/detail` | required | 30/minute per user |
| `POST` | `/puzzle/guess` | required | 10/minute per user |
| `POST` | `/music` | admin key | 20/hour per IP |
| `GET` | `/music` | none | 30/minute per IP |
| `POST` | `/admin/cache/clear` | admin key | 20/hour per IP |
| `POST` | `/admin/user/delete` | admin key | 20/hour per IP |

---

### `GET /health`

Reports the health of every backing dependency (MySQL, PostgreSQL,
Redis, MongoDB). Not authenticated, not rate-limited.

**curl**
```bash
curl --location 'http://localhost:8080/health'
```

**Response `200 OK`** — all dependencies reachable:
```json
{
  "status": "ok"
}
```

**Response `503 Service Unavailable`** — one or more dependencies down:
```json
{
  "status": "unhealthy",
  "errors": {
    "mysql": "dial tcp 127.0.0.1:3306: connect: connection refused"
  }
}
```

| Field | Type | Meaning |
|---|---|---|
| `status` | string | `"ok"` or `"unhealthy"`. |
| `errors` | object (optional) | Present only when unhealthy. One key per failed dependency (`mysql`, `postgresql`, `redis`, `mongodb`), value is the underlying connection error. Absent keys mean that dependency is healthy. |

---

### `POST /users/signup`

Creates a new account. On success, an auth token is issued via the
`X-Auth-Token` response header — the account starts **unverified**; call
`/users/verify/initiate` and `/users/verify/decide` afterward to verify
it.

**Request body** — `SignUpRequest`

| Field | Type | Required | Notes |
|---|---|---|---|
| `email` | string | yes | Normalized (trimmed + lowercased) server-side; must be a valid, addr-only email (no display name). |
| `password` | string | yes | 8-32 ASCII characters; must contain at least one uppercase letter, one lowercase letter, one digit, and one punctuation/symbol character. |

**curl**
```bash
curl --location 'http://localhost:8080/users/signup' \
--header 'Content-Type: application/json' \
--data '{
    "email": "player@example.com",
    "password": "SecurePass123!"
}'
```

**Response `201 Created`**
```json
{
  "message": "Account created successfully"
}
```
Response headers include `X-Auth-Token: <encrypted email>` — save this.

**Possible errors**

| Status | `error_message` | Cause |
|---|---|---|
| 400 | *(raw JSON parse error)* | Malformed request body. `show_response_as_is: false`. |
| 400 | "Please enter a valid email address." | `email` failed validation. |
| 400 | *(one of the specific password rules, e.g. "Password must contain at least one uppercase letter.")* | `password` failed a specific complexity check. |
| 409 | "An account with this email already exists. Try logging in instead." | Email already registered. |
| 429 | "Too many requests. Please slow down and try again shortly." | Rate limit exceeded. |

---

### `POST /users/login`

Authenticates an existing account and issues a fresh auth token, whether
or not the account is verified. The response body's `is_verified` reflects
the account's actual verification status, so a client can route a login
straight to the right place without waiting on a separate call.

**Request body** — `LoginRequest`

| Field | Type | Required | Notes |
|---|---|---|---|
| `email` | string | yes | |
| `password` | string | yes | Compared using a constant-time comparison. |

**curl**
```bash
curl --location 'http://localhost:8080/users/login' \
--header 'Content-Type: application/json' \
--data '{
    "email": "player@example.com",
    "password": "SecurePass123!"
}'
```

**Response `200 OK`**
```json
{
  "message": "Logged in successfully",
  "is_verified": true
}
```
Response headers include `X-Auth-Token: <encrypted email>`.

**Possible errors**

| Status | `error_message` | Cause |
|---|---|---|
| 400 | *(raw JSON parse error)* | Malformed body. |
| 400 | "Please enter a valid email address." | `email` failed validation. |
| 401 | "The email or password you entered is incorrect." | Wrong password, **or** no account with that email — deliberately identical for both cases so the API can't be used to discover which emails are registered. |
| 429 | "Too many requests. Please slow down and try again shortly." | Rate limit exceeded. |

---

### `POST /users/verify/initiate`

Sends a one-time verification code (5-6 characters) to the authenticated
user's email (valid for 5 minutes). **Requires auth.**

**Request body:** none.

**curl**
```bash
curl --location --request POST 'http://localhost:8080/users/verify/initiate' \
--header 'X-Auth-Token: <token from signup/login>'
```

**Response `200 OK`**
```json
{
  "message": "Verification email sent"
}
```

**Possible errors**

| Status | `error_message` | Cause |
|---|---|---|
| 401 | "Please LogIn or SignUp" | Missing/invalid `X-Auth-Token`. |
| 404 | "We couldn't find an account with that email." | Should not normally occur if the token is valid, but surfaced if the underlying account is gone. |
| 409 | "This account is already verified." | Account is already verified — no new code is sent. |
| 429 | "Too many requests. Please slow down and try again shortly." | Rate limit exceeded (this endpoint sends real email, so it's the tightest-capped one). |

---

### `POST /users/verify/decide`

Submits the OTP received by email to complete verification. **Requires
auth.**

**Request body** — `decideVerificationRequest`

| Field | Type | Required | Notes |
|---|---|---|---|
| `otp` | string | yes | The code sent by `/users/verify/initiate`. Compared using a constant-time comparison. |

**curl**
```bash
curl --location 'http://localhost:8080/users/verify/decide' \
--header 'X-Auth-Token: <token from signup/login>' \
--header 'Content-Type: application/json' \
--data '{
    "otp": "A1B2C3"
}'
```

**Response `200 OK`**
```json
{
  "message": "Account verified successfully"
}
```

**Possible errors**

| Status | `error_message` | Cause |
|---|---|---|
| 401 | "Please LogIn or SignUp" | Missing/invalid `X-Auth-Token`. |
| 400 | *(raw JSON parse error)* | Malformed body. |
| 400 | "That verification code is invalid or has expired. Please request a new one." | Wrong OTP, or the 5-minute window elapsed — deliberately identical wording for both, and a wrong OTP does **not** consume/invalidate the correct one, so the user can retry until it expires. |
| 404 | "We couldn't find an account with that email." | See above. |
| 429 | "Too many requests. Please slow down and try again shortly." | Rate limit exceeded. |

---

### `POST /puzzle`

Generates and creates a new puzzle for the authenticated user, then
returns their full, refreshed puzzle list. **Requires auth and a
verified account.** Difficulty is calibrated automatically from the
user's win rate and average similarity score across their most recent
puzzles (not their whole lifetime history).

This is the most expensive endpoint (three LLM/embedding calls per
request), hence the tightest rate limit.

**Request body:** none.

**curl**
```bash
curl --location --request POST 'http://localhost:8080/puzzle' \
--header 'X-Auth-Token: <token from signup/login>'
```

**Response `201 Created`** — array of `PuzzleElement`, every puzzle the
user has (including the newly-created one):
```json
[
  {
    "puzzle_id": "01J8XYZABC0000000000000ZZ",
    "canonical_answer": "42"
  }
]
```

| Field | Type | Meaning |
|---|---|---|
| `puzzle_id` | string | ULID identifying the puzzle — use this for `/puzzle/detail` and `/puzzle/guess`. |
| `canonical_answer` | string | The AI's terse answer — this is the clue; it's never a secret. |

**Possible errors**

| Status | `error_message` | Cause |
|---|---|---|
| 401 | "Please LogIn or SignUp" | Missing/invalid `X-Auth-Token`. |
| 403 | "Please verify your email before continuing." | Account exists but isn't verified yet. |
| 429 | "Too many requests. Please slow down and try again shortly." | Rate limit exceeded. |
| 500 | *(internal error)* | An upstream AI/embedding call failed. `show_response_as_is: false`. |

---

### `GET /puzzle`

Lists every puzzle belonging to the authenticated user. **Requires auth
and a verified account** (same check as `POST /puzzle`).

**curl**
```bash
curl --location 'http://localhost:8080/puzzle' \
--header 'X-Auth-Token: <token from signup/login>'
```

**Response `200 OK`** — same `PuzzleElement[]` shape as `POST /puzzle`
above (empty array `[]` if the user has no puzzles yet).

**Possible errors**

| Status | `error_message` | Cause |
|---|---|---|
| 401 | "Please LogIn or SignUp" | Missing/invalid `X-Auth-Token`. |
| 403 | "Please verify your email before continuing." | Account exists but isn't verified yet. |
| 429 | "Too many requests. Please slow down and try again shortly." | Rate limit exceeded. |

---

### `GET /puzzle/detail`

Fetches full details for one puzzle, including its message history. The
caller must own the puzzle. **Requires auth.**

**Query parameters**

| Name | Required | Notes |
|---|---|---|
| `puzzle_id` | yes | The ULID from `PuzzleElement.puzzle_id`. |

**curl**
```bash
curl --location 'http://localhost:8080/puzzle/detail?puzzle_id=01J8XYZABC0000000000000ZZ' \
--header 'X-Auth-Token: <token from signup/login>'
```

**Response `200 OK`** — `PuzzleDetails`:
```json
{
  "puzzle_id": "01J8XYZABC0000000000000ZZ",
  "canonical_answer": "42",
  "messages": [
    { "role": "user", "message": "What is the meaning of life?", "timestamp": "2026-07-22T10:15:00Z" },
    { "role": "assistant", "message": "Getting warmer, but think smaller and more literal.", "timestamp": "2026-07-22T10:15:02Z" }
  ],
  "max_intent_similarity_percentage": 41,
  "user_win_status": false,
  "created_at": "2026-07-22T10:14:50Z",
  "updated_at": "2026-07-22T10:15:02Z"
}
```

| Field | Type | Meaning |
|---|---|---|
| `puzzle_id` | string | ULID. |
| `canonical_answer` | string | The AI's terse answer — always shown; it's the clue, not a secret. |
| `hidden_prompt` | string (omitted until won) | The actual hidden prompt — **only present once `user_win_status` is `true`**. Absent entirely (not even an empty string key) while still unsolved. |
| `messages` | array of `MessageElement` | Full guess/hint history, sorted oldest-to-newest. `role` is `"user"` (a guess) or `"assistant"` (a hint). |
| `max_intent_similarity_percentage` | int (0-100) | The best similarity score achieved across all guesses so far. |
| `user_win_status` | bool | Whether this puzzle has been solved. |
| `latest_guess_metrics` | object (omitted here) | Only populated by `POST /puzzle/guess`'s response — see below. Always absent on a plain read since per-guess breakdowns aren't persisted. |
| `created_at` / `updated_at` | RFC3339 timestamp | |

**Possible errors**

| Status | `error_message` | Cause |
|---|---|---|
| 401 | "Please LogIn or SignUp" | Missing/invalid `X-Auth-Token`. |
| 400 | "That puzzle ID doesn't look right." | `puzzle_id` missing or not a valid ULID. |
| 404 | "We couldn't find that puzzle." | No such puzzle, **or** it exists but belongs to a different user — deliberately identical so this endpoint can't be used to enumerate other users' puzzle IDs. |
| 429 | "Too many requests. Please slow down and try again shortly." | Rate limit exceeded. |

---

### `POST /puzzle/guess`

Submits a guess at the hidden prompt. This is the core gameplay action:
the guess is scored, the puzzle's best-score-so-far is updated if this
guess beat it, a win is recorded if the score clears the win threshold,
and — if it wasn't a win — a hint is generated and appended to the
conversation. **Requires auth. Rejects further guesses once the puzzle
is already won.**

**Request body** — `addUserPromptRequest`

| Field | Type | Required | Notes |
|---|---|---|---|
| `puzzle_id` | string | yes | Must be a valid ULID for a puzzle you own. |
| `user_prompt` | string | yes | Your guess at the hidden prompt. Non-empty after trimming whitespace. |

**curl**
```bash
curl --location 'http://localhost:8080/puzzle/guess' \
--header 'X-Auth-Token: <token from signup/login>' \
--header 'Content-Type: application/json' \
--data '{
    "puzzle_id": "01J8XYZABC0000000000000ZZ",
    "user_prompt": "What is forty two in numeric"
}'
```

**Response `200 OK`** — `PuzzleDetails`, same shape as `GET /puzzle/detail`,
but with `latest_guess_metrics` populated for the guess you just made:
```json
{
  "puzzle_id": "01J8XYZABC0000000000000ZZ",
  "canonical_answer": "42",
  "hidden_prompt": "What is forty two in numeric",
  "messages": [
    { "role": "user", "message": "What is forty two in numeric", "timestamp": "2026-07-22T10:16:40Z" }
  ],
  "max_intent_similarity_percentage": 96,
  "user_win_status": true,
  "latest_guess_metrics": {
    "embedding_dot_product": 68.42,
    "cosine_similarity_score": 92.14,
    "llm_similarity_score": 98,
    "final_similarity_score": 95.66,
    "considered_won": true
  },
  "created_at": "2026-07-22T10:14:50Z",
  "updated_at": "2026-07-22T10:16:40Z"
}
```

`latest_guess_metrics` fields:

| Field | Type | Meaning |
|---|---|---|
| `embedding_dot_product` | float | Raw (non-normalized) dot product between your guess's embedding and the hidden prompt's embedding (informational only — not part of the score; unlike cosine similarity, this is sensitive to embedding magnitude, not just direction). |
| `cosine_similarity_score` | float (0-100) | Embedding cosine similarity, scaled to a percentage. |
| `llm_similarity_score` | float (0-100) | The LLM's own judged intent-match score. `0` if the cosine score was too low to bother calling the LLM. |
| `final_similarity_score` | float (0-100) | The blended score (weighted cosine + LLM) that actually determines win/loss and updates `max_intent_similarity_percentage`. |
| `considered_won` | bool | Whether **this specific guess** cleared the win threshold. |

Note: when `user_win_status` becomes `true`, `hidden_prompt` is now
present in the response (see the example above) and no assistant hint is
generated for this guess — there's nothing left to hint at.

**Possible errors**

| Status | `error_message` | Cause |
|---|---|---|
| 401 | "Please LogIn or SignUp" | Missing/invalid `X-Auth-Token`. |
| 400 | "That puzzle ID doesn't look right." | `puzzle_id` missing/invalid. |
| 400 | "Please enter a guess before submitting." | `user_prompt` empty/whitespace-only. |
| 404 | "We couldn't find that puzzle." | No such puzzle, or it isn't yours. |
| 409 | "You've already solved this puzzle!" | The puzzle was already won — no further guesses accepted. |
| 429 | "Too many requests. Please slow down and try again shortly." | Rate limit exceeded. |
| 500 | *(internal error)* | An upstream AI/embedding call failed. `show_response_as_is: false`. |

---

### `POST /music`

Inserts one or more music catalog entries, then returns the full,
refreshed catalog. **Requires `X-Admin-Key`.** Entries whose title already
exists in the catalog (case/whitespace-insensitive match) are silently
skipped rather than duplicated or erroring the whole batch.

**Request body:** a bare JSON array of entries to insert. `id` and
`created_at` are server-generated — omit them (they're ignored if sent).

| Field | Type | Required | Notes |
|---|---|---|---|
| `title` | string | yes | Must be non-empty after trimming. |
| `artists` | string[] | no | |
| `genre` | string[] | no | |
| `wave_audio_link` | string | yes | Must be a valid URL. |
| `artwork_link` | string | yes | Must be a valid URL. |

**curl**
```bash
curl --location 'http://localhost:8080/music' \
--header 'X-Admin-Key: <the configured admin key>' \
--header 'Content-Type: application/json' \
--data '[
  {
    "title": "Neon Skyline",
    "artists": ["Synth Rider"],
    "genre": ["synthwave"],
    "wave_audio_link": "https://cdn.example.com/audio/neon-skyline.wav",
    "artwork_link": "https://cdn.example.com/art/neon-skyline.png"
  }
]'
```

**Response `201 Created`** — array of `MusicDetail`, the entire catalog
(including everything inserted before this call, not just what this call
added):
```json
[
  {
    "id": "665f1a2b3c4d5e6f7a8b9c0d",
    "title": "Neon Skyline",
    "artists": ["Synth Rider"],
    "genre": ["synthwave"],
    "wave_audio_link": "https://cdn.example.com/audio/neon-skyline.wav",
    "artwork_link": "https://cdn.example.com/art/neon-skyline.png",
    "created_at": "2026-07-24T04:50:05Z"
  }
]
```

| Field | Type | Meaning |
|---|---|---|
| `id` | string | Server-generated catalog entry ID. |
| `title` | string | |
| `artists` | string[] | |
| `genre` | string[] | |
| `wave_audio_link` | string | |
| `artwork_link` | string | |
| `created_at` | string (RFC3339) | |

**Possible errors**

| Status | `error_message` | Cause |
|---|---|---|
| 401 | "Invalid or missing admin key." | Missing/incorrect `X-Admin-Key`. |
| 400 | "Please provide at least one music detail to save." | Request body is an empty array. |
| 400 | *(raw JSON parse error)* | Malformed body. |
| 429 | "Too many requests. Please slow down and try again shortly." | Rate limit exceeded. |

---

### `GET /music`

Lists the entire music catalog. **No authentication required at all** —
this is the one endpoint in this API callable with no headers.

**curl**
```bash
curl --location 'http://localhost:8080/music'
```

**Response `200 OK`** — same `MusicDetail[]` shape as `POST /music` above
(empty array `[]` if the catalog is empty). Served from an in-memory
cache, then Redis, falling back to the database — see
[README.md](./README.md#observability) for the caching layers.

**Possible errors**

| Status | `error_message` | Cause |
|---|---|---|
| 429 | "Too many requests. Please slow down and try again shortly." | Rate limit exceeded. |

---

### `POST /admin/cache/clear`

Flushes every ephemeral cache entry (OTPs, rate-limit counters — nothing
that's a system of record). **Requires `X-Admin-Key`.**

**Request body:** none.

**curl**
```bash
curl --location --request POST 'http://localhost:8080/admin/cache/clear' \
--header 'X-Admin-Key: <the configured admin key>'
```

**Response `200 OK`**
```json
{ "message": "Cache cleared" }
```

**Possible errors**

| Status | `error_message` | Cause |
|---|---|---|
| 401 | "Invalid or missing admin key." | Missing/incorrect `X-Admin-Key`. |
| 429 | "Too many requests. Please slow down and try again shortly." | Rate limit exceeded. |

---

### `POST /admin/user/delete`

Permanently deletes a user's account and their puzzles. **Requires
`X-Admin-Key`.**

**Request body**

| Field | Type | Required | Notes |
|---|---|---|---|
| `email` | string | yes | |

**curl**
```bash
curl --location 'http://localhost:8080/admin/user/delete' \
--header 'X-Admin-Key: <the configured admin key>' \
--header 'Content-Type: application/json' \
--data '{
    "email": "player@example.com"
}'
```

**Response `200 OK`**
```json
{ "message": "User data deleted" }
```

**Possible errors**

| Status | `error_message` | Cause |
|---|---|---|
| 401 | "Invalid or missing admin key." | Missing/incorrect `X-Admin-Key`. |
| 429 | "Too many requests. Please slow down and try again shortly." | Rate limit exceeded. |

---

## Unmatched routes

Any request that doesn't match a route above gets:

- **`404 Not Found`** — `{"error_message": "Not Found"}` — for an unknown path.
- **`405 Method Not Allowed`** — `{"error_message": "Method Not Allowed"}` — for a known path called with the wrong HTTP method.
