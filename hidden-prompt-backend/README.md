# Hidden Prompt — Reverse Prompt Intelligence (RPI)

Hidden Prompt is a guessing game with a twist: instead of asking an AI a
question and reading its answer, you see only the AI's terse, one-line
**answer** and have to reverse-engineer the *question that produced it*.

> The AI says **"42"**. What did the player ask it?
> (`What is forty two in numeric`)

This repository is the Go backend that powers the game: puzzle generation,
answer scoring, hints, accounts, and everything else needed to serve the
game over HTTP.

For the full REST API reference (every endpoint, request/response shapes,
and ready-to-run curl examples), see **[API.md](./API.md)**.

---

## Table of Contents

- [How the game works](#how-the-game-works)
- [Lifecycle & flow diagrams](#lifecycle--flow-diagrams)
  - [Request lifecycle](#request-lifecycle)
  - [Account lifecycle](#account-lifecycle)
  - [Puzzle creation flow](#puzzle-creation-flow)
  - [Guess scoring decision tree](#guess-scoring-decision-tree)
  - [Hint generation decision tree](#hint-generation-decision-tree)
  - [Puzzle lifecycle](#puzzle-lifecycle)
  - [Layered architecture](#layered-architecture)
- [Architecture](#architecture)
- [Tech stack](#tech-stack)
- [Project layout](#project-layout)
- [Getting started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Configuration](#configuration)
  - [Running the server](#running-the-server)
  - [Running tests](#running-tests)
- [Authentication](#authentication)
- [Rate limiting](#rate-limiting)
- [Error handling conventions](#error-handling-conventions)
- [Observability](#observability)
- [License](#license)

---

## How the game works

1. **Puzzle generation.** When a player starts a new puzzle, the backend
   asks an LLM (Groq) in a single call for both a short, non-academic
   question — the *hidden prompt* — and its own terse answer to it (think
   "42", not "The answer to your question is forty-two") — the *canonical
   answer*, the only thing shown to the player up front. Difficulty is
   calibrated to that player's skill level (derived from their win rate and
   average similarity score across their most recent puzzles, not their
   whole lifetime history, so a rough start or a long streak doesn't
   permanently anchor their difficulty tier), and the question's content
   domain is picked from a weighted style pool so puzzles don't all cluster
   around the same handful of topics. The model is also shown the player's
   own recent questions/answers and asked to avoid repeating them. Once a
   well-formed pair comes back, the backend separately asks Gemini for an
   embedding of the hidden prompt, stored for scoring later guesses.
2. **Guessing.** The player sees the canonical answer and submits guesses
   at what question produced it. Each guess is scored for *intent*
   similarity to the real hidden prompt — not exact wording — using a
   blend of:
   - **Cosine similarity** between the guess's embedding and the hidden
     prompt's embedding (cheap, fast, computed first as an early-reject
     filter).
   - **LLM-judged intent similarity** (only computed if the cosine score
     clears a minimum bar, to save API calls on wildly-off guesses).
   - **Embedding dot product**, reported alongside the score for extra
     signal.
3. **Hints.** If a guess doesn't win, the backend generates a hint via the
   LLM, calibrated to the player's skill tier and given the full guess
   history for that puzzle — without ever revealing the hidden prompt or
   canonical answer outright.
4. **Winning.** Once a guess's blended similarity score crosses the win
   threshold, the puzzle is marked won, the hidden prompt is revealed in
   the API response, and no further guesses are accepted for that puzzle.

## Lifecycle & flow diagrams

Every diagram below reflects the actual code paths in this repo (function
and file names included) rather than a simplified sketch — if the code
changes, these should change with it.

### Request lifecycle

Every HTTP request, regardless of endpoint, passes through the same
middleware → rate-limit → auth → handler → service → repository chain.
`PanicRecovery` is the outermost layer so nothing — including a panic
inside CORS handling itself — can escape uncaught
(`pkg/core/router/router.go`: `mw.PanicRecovery(mw.AllowCORS(fr.Handler))`).

```mermaid
sequenceDiagram
    participant Client
    participant PanicRecovery as Middleware: PanicRecovery
    participant CORS as Middleware: AllowCORS
    participant Router as fasthttp/router
    participant Handler
    participant Service
    participant Repo as Repository (MySQL / PostgreSQL / Redis)

    Client->>PanicRecovery: HTTP request
    PanicRecovery->>CORS: next(ctx)
    CORS->>CORS: set CORS headers
    alt method is OPTIONS
        CORS-->>Client: 204 (preflight answered directly)
    end
    CORS->>Router: next(ctx)
    Router->>Handler: route matched
    Handler->>Handler: timeTakenStart()
    Handler->>Handler: enforceRateLimit() via Redis GCRA
    alt over the limit
        Handler-->>Client: 429 + Retry-After header
    end
    Handler->>Handler: requireAuth() decrypts X-Auth-Token
    alt token missing or invalid
        Handler-->>Client: 401 "Please LogIn or SignUp"
    end
    Handler->>Service: bound request DTO
    Service->>Repo: query / mutate
    Repo-->>Service: data or typed error
    Service-->>Handler: result or typed error
    alt error
        Handler->>Handler: writeServiceError() maps error to status+message
    else success
        Handler->>Handler: writeJSONResponse()
    end
    Handler-->>Client: JSON body + X-Time-Taken header
    Note over PanicRecovery: a panic anywhere above is recovered here,<br/>logged with a stack trace, turned into a clean 500
```

### Account lifecycle

There's no session store — identity is an AES-SIV-encrypted email in the
`X-Auth-Token` header, minted on signup/login and decrypted on every
protected request (`pkg/handler/helper.go`: `setAuth` / `getAuth`).
Verification is a separate, independent flag on the account, checked only
where it matters (creating a puzzle).

```mermaid
stateDiagram-v2
    [*] --> Unauthenticated
    Unauthenticated --> Unverified: signup, or login to an unverified account
    Unauthenticated --> Verified: login to an already-verified account
    Unverified --> OTPSent: verify/initiate
    OTPSent --> Verified: verify/decide, correct OTP
    OTPSent --> OTPSent: verify/decide, wrong or expired OTP
    Verified --> [*]: can call POST /puzzle
```

Details that don't fit on the arrows above: `POST /users/verify/initiate`
generates a random 5-6 character OTP, stores it in Redis with a TTL
(`constants.OTPValidity`), and emails it — calling it again on an
already-verified account is rejected with `409 UserAlreadyVerified` rather
than sending a needless email. `POST /users/verify/decide` compares the
submitted OTP against Redis with a constant-time comparison
(`crypto/subtle`), same as password checks on login.

### Puzzle creation flow

The question and its answer come back from a single `GeneratePuzzle` call
rather than two separate ones — asking the model for both at once keeps
them consistent with each other and halves the LLM round-trips per puzzle
(`pkg/service/puzzle/puzzle.go`: `CreatePuzzleForUser`). `GenerateEmbeddings`
(Gemini) runs afterward, sequentially — it needs the finished hidden
prompt text as its input, so there's nothing to parallelize. The
generation call is retry-guarded (up to 3 attempts) against a model
ignoring its formatting instructions or returning a malformed
question/answer (`pkg/chat/chat.go`).

```mermaid
sequenceDiagram
    participant Handler
    participant PuzzleSvc as puzzle.Service
    participant UserSvc as user.Service
    participant AI as chat.AI (Groq + Gemini)
    participant DB as PostgreSQL (puzzles)

    Handler->>PuzzleSvc: CreatePuzzleForUser(email)
    PuzzleSvc->>UserSvc: IsVerified(email)
    alt not verified
        PuzzleSvc-->>Handler: 403 UserIsNotVerified
    end
    PuzzleSvc->>DB: GetPuzzlesByEmail (recent history, win-rate + avg-score, all-time answers)
    PuzzleSvc->>PuzzleSvc: deriveDifficultyTier + choosePuzzleStyle
    PuzzleSvc->>AI: GeneratePuzzle(tier, style, recentPuzzles, usedAnswers)
    loop up to 3 attempts
        AI->>AI: Groq call (GPT-OSS-20B or Llama 3.3 70B)
        AI->>AI: isMalformedGeneratedQuestion / isMalformedGeneratedAnswer?<br/>(empty / over 40-or-12 words / newline / "**")
    end
    AI-->>PuzzleSvc: {hidden prompt, canonical answer}

    PuzzleSvc->>AI: GenerateEmbeddings(hiddenPrompt)
    AI-->>PuzzleSvc: embedding vector

    PuzzleSvc->>DB: CreatePuzzle(hiddenPrompt, canonicalAnswer, embedding)
    PuzzleSvc-->>Handler: [{puzzle_id, canonical_answer}, ...]
```

If all 3 attempts come back malformed, puzzle creation fails outright
(`ErrAllAttemptsMalformed`) rather than persisting a broken puzzle —
unlike a hint, a malformed hidden prompt or canonical answer would poison
that puzzle for its entire lifetime.

### Guess scoring decision tree

`pkg/service/intent/intent.go`: `GetSimilarityMetrics`. The self-consistency
recheck exists because a single LLM sample is occasionally just noise — a
guess with zero real connection to a puzzle once scored a perfect 100 from
one weak-model draw despite unremarkable embedding similarity. There are
three cases that trigger a confirmation call: a suspiciously high LLM
score with weak cosine support (take the lower of the two draws), a
suspiciously low LLM score with strong cosine support (take the higher),
and — since neither of those catches an entirely ordinary-looking score
that's simply noisy right around the win threshold, which is exactly the
range where that noise actually flips a win into a loss — a blended
estimate landing within 5 points of the win threshold either way (take the
average of the two draws, since there's no cosine-based signal for which
one to trust more in that case).

```mermaid
flowchart TD
    A["AddUserPrompt: new guess"] --> B{"Puzzle already won?"}
    B -- yes --> B1["Reject: 409 PuzzleAlreadyWon"]
    B -- no --> C["Append guess to message history"]
    C --> D{"Normalized guess equals<br/>normalized hidden prompt?"}
    D -- yes --> E["Perfect score: 100 / 100 / 100<br/>ConsideredWon = true"]
    D -- no --> F["Generate embedding for the guess"]
    F --> G["Cosine similarity vs the puzzle's<br/>stored hidden-prompt embedding"]
    G --> H{"cosine under 30?"}
    H -- yes --> I["final = cosine, as-is<br/>LLM is never called<br/>ConsideredWon = false"]
    H -- no --> J["ScoreIntent #1: LLM judges guess vs hidden prompt<br/>0-29 unrelated/meta &middot; 30-59 same topic, wrong question<br/>60-89 same question, missing the key detail &middot; 90-100 same fact"]
    J --> K1{"score 85 or higher,<br/>AND cosine under 70?"}
    K1 -- yes --> M1["confirmation call"] --> N1["use the LOWER of the two scores"]
    K1 -- no --> K2{"score under 60,<br/>AND cosine 85 or higher?"}
    K2 -- yes --> M2["confirmation call"] --> N2["use the HIGHER of the two scores"]
    K2 -- no --> K3{"blended estimate within<br/>5 points of the win threshold?"}
    K3 -- yes --> M3["confirmation call"] --> N3["use the AVERAGE of the two scores"]
    K3 -- no --> L["use LLM score as-is"]
    N1 --> O["final = 0.40 &times; cosine + 0.60 &times; llmScore"]
    N2 --> O
    N3 --> O
    L --> O
    O --> P{"final at least 90?"}
    P -- yes --> Q["ConsideredWon = true<br/>hidden_prompt now revealed"]
    P -- no --> R["ConsideredWon = false<br/>generate a hint"]
```

### Hint generation decision tree

`pkg/chat/chat.go`: `GenerateHint`. Two independent output-side checks run
on every attempt — a hint is only ever returned if it passes both.

```mermaid
flowchart TD
    A["Losing guess needs a hint"] --> B["Build prompt: hidden prompt, canonical answer,<br/>last 12 turns, this guess,<br/>current score vs previous best, skill tier"]
    B --> C["Groq call, temperature capped at 1.1"]
    C --> D{"revealsSecret?<br/>'the answer is X' always flags &middot;<br/>bare word X flags UNLESS<br/>the player's own guess already said X"}
    D -- leaked --> E["retry (up to 3 attempts total)"]
    D -- clean --> F{"isUnusableHint?<br/>refusal phrase (curly or straight apostrophe)<br/>or over 40 words"}
    F -- unusable --> E
    F -- usable --> G["Return the hint to the player"]
    E --> H{"attempts exhausted?"}
    H -- no --> C
    H -- "yes, last one leaked" --> I["Fallback: 'Getting warmer -<br/>think about what makes this one specific.'"]
    H -- "yes, last one was a refusal" --> J["Fallback: 'About the same as last time -<br/>think it through and try a different angle.'"]
```

### Puzzle lifecycle

The full state arc of a single puzzle, tying the two decision trees above
together.

```mermaid
stateDiagram-v2
    [*] --> Created: POST /puzzle
    Created --> Guessing: player sees canonical_answer only
    Guessing --> Guessing: score under 90
    Guessing --> Won: score 90+
    Won --> [*]: further guesses rejected with 409
```

Every non-winning guess in the `Guessing` loop appends a hint to the
message history and, if its score beats the puzzle's previous best,
updates `max_intent_similarity_percentage`. Landing in `Won` reveals
`hidden_prompt` in that response for the first and only time.

### Layered architecture

A pictorial companion to the [dependency table](#architecture) below — same
layering, drawn as a graph instead of a stack.

```mermaid
flowchart TB
    Config["pkg/config<br/>encrypted, embedded configuration"]
    Clients["pkg/clients/groq, gemini<br/>outbound LLM / embedding clients"]
    DB["pkg/database/mysql, psql, redis<br/>users, puzzles, OTP + rate-limit state"]
    Algo["pkg/algo<br/>cosine similarity + embedding dot product"]
    Chat["pkg/chat<br/>AI layer: prompt generation, hints, intent scoring"]
    Services["pkg/service/user, intent, puzzle<br/>business logic"]
    Handlers["pkg/handler<br/>HTTP handlers (fasthttp)"]
    Middleware["pkg/core/middleware<br/>CORS + panic recovery"]
    Router["pkg/core/router<br/>route table"]
    Deps["pkg/core/deps<br/>fx dependency wiring + server lifecycle"]

    Config --> Clients
    Config --> DB
    Clients --> Chat
    DB --> Services
    Algo --> Services
    Chat --> Services
    Services --> Handlers
    Handlers --> Router
    Middleware --> Router
    Router --> Deps
```

## Architecture

The codebase is layered strictly bottom-up — each layer only depends on the
one below it, and every constructor nil-checks its dependencies:

```
pkg/config           encrypted, embedded configuration (see Configuration below)
pkg/clients/{groq,gemini}   outbound LLM/embedding API clients
pkg/database/{mysql,psql,redis}   users (MySQL), puzzles (PostgreSQL + pgvector), OTP/rate-limit state (Redis)
pkg/algo             cosine similarity + embedding dot product
pkg/chat             the AI layer: prompt generation, hints, intent scoring (talks to clients, not to the database)
pkg/service/{user,intent,puzzle,music,admin,cron}   business logic, composed from the layers above
pkg/handler          HTTP handlers (fasthttp): binds requests, calls services, maps errors to responses
pkg/core/middleware  CORS + panic recovery, wrapping the whole router
pkg/core/router      route table (github.com/fasthttp/router)
pkg/core/deps        dependency-injection wiring (uber-go/fx) + the fasthttp server lifecycle
```

Every package is behind an interface with an `New*` constructor, wired
together in `pkg/core/deps`. The entrypoint is the `main.go` at the repo
root — see [Running the server](#running-the-server).

## Tech stack

| Concern | Choice |
|---|---|
| Language | Go 1.26 |
| HTTP server | [`valyala/fasthttp`](https://github.com/valyala/fasthttp) |
| Router | [`fasthttp/router`](https://github.com/fasthttp/router) |
| Dependency injection | [`uber-go/fx`](https://github.com/uber-go/fx) |
| User accounts | MySQL |
| Puzzles | PostgreSQL + [`pgvector`](https://github.com/pgvector/pgvector) (for the hidden-prompt embedding column) |
| Music catalog | MongoDB |
| OTP storage / rate limiting | Redis (rate limiting via [`go-redis/redis_rate`](https://github.com/go-redis/redis_rate), a GCRA/token-bucket limiter) |
| LLM (prompt/answer/hint generation, intent scoring) | [Groq](https://groq.com) |
| Embeddings | [Gemini](https://ai.google.dev) |
| Outbound email (OTP delivery) | SMTP, load-balanced across configured accounts |
| Dynamic prompt templating | [`valyala/quicktemplate`](https://github.com/valyala/quicktemplate) (`.qtpl` files, compiled via `qtc`) |

## Project layout

```
pkg/
  algo/            cosine similarity + embedding dot product
  chat/            AI layer (hidden prompt / canonical answer / hint / intent-score generation)
    internal/prompts/  quicktemplate-generated dynamic user-prompt builders
    templates/         static system prompts (.md, embedded at build time)
  clients/
    groq/          Groq chat-completion client
    gemini/        Gemini embeddings client
  config/          encrypted configuration loading
  constants/       shared string/const values (header names, rate-limit keys, ...)
  core/
    deps/          fx dependency-injection wiring + server lifecycle
    middleware/     CORS + panic recovery
    router/        route table
  database/
    mysql/users/   user accounts (sqlc-generated queries)
    psql/puzzles/  puzzles (sqlc-generated queries, pgvector embedding column)
    mongo/music/   music catalog (unique index on normalized title)
    redis/         thin Redis repository wrapper
  errors/          typed, user-facing error wrappers shared across services
  handler/         HTTP handlers (user, puzzle, music, admin, health)
  mail/            OTP email sending
  models/          request/response DTOs shared between services and handlers
  serialization/    generic struct -> compact text encoder (used to keep LLM prompts token-cheap)
  service/
    user/          signup, login, email verification
    intent/        guess-scoring (cosine + LLM + dot product blend)
    puzzle/        puzzle CRUD + the guess/hint/win game loop
    music/         music catalog, with an in-memory + Redis read-through cache
    admin/         admin-key-gated cache clear / user deletion
    cron/          DB keep-alive ping every 5 minutes (pkg/constants/cron.go: PingCronTime)
  smtp/            low-level SMTP sending
  utils/           email/password validation, crypto, ULIDs, JSON extraction from LLM text, ...
```

## Getting started

### Prerequisites

- Go 1.26+
- A MySQL instance (user accounts)
- A PostgreSQL instance with the `vector` extension available (the puzzles
  repository creates it automatically on startup if missing)
- A MongoDB instance (music catalog)
- A Redis instance
- Groq and Gemini API keys
- An SMTP account (or several) for sending verification OTP emails
- [`qtc`](https://github.com/valyala/quicktemplate) if you need to
  regenerate the `.qtpl.go` files under `pkg/chat/internal/prompts`

### Configuration

Configuration is **not** read from plain environment variables or a
`.env` file. Each config section (`groq_keys.txt`, `gemini_keys.txt`,
`mysql.txt`, `postgresql.txt`, `mongodb.txt`, `redis.txt`, `mail.txt` under
`pkg/config/`) is a base64-encoded, AES-256-GCM-encrypted JSON blob,
embedded into the binary at build time via `//go:embed` and decrypted at
startup.

The decryption key comes from the `CONFIG_DECODING_KEY` environment
variable (base64-encoded, 32 raw bytes for AES-256). **If unset, a
hardcoded development key is used as a fallback** — this is fine for local
development against the checked-in dev config, but `CONFIG_DECODING_KEY`
**must** be set to a real secret in any shared or production environment.

Each blob decodes to a JSON object matching one of these shapes:

<details>
<summary><code>pkg/config/mysql.txt</code></summary>

```json
{
  "mysql_config": {
    "host": "127.0.0.1",
    "port": 3306,
    "username": "root",
    "database": "hidden_prompt",
    "password": "...",
    "insecure_tls_skip_verify": true,
    "ca_cert_string": ""
  }
}
```
`ca_cert_string` (a PEM-encoded CA certificate) is required whenever
`insecure_tls_skip_verify` is `false`.
</details>

<details>
<summary><code>pkg/config/postgresql.txt</code></summary>

```json
{
  "postgre_sql_config": {
    "host": "127.0.0.1",
    "port": 5432,
    "username": "postgres",
    "database": "hidden_prompt",
    "password": "...",
    "insecure_tls_skip_verify": true,
    "ca_cert_string": ""
  }
}
```
</details>

<details>
<summary><code>pkg/config/redis.txt</code></summary>

```json
{
  "redis_config": {
    "addrs": ["127.0.0.1:6379"],
    "username": "",
    "password": "",
    "database": 0,
    "insecure_tls_skip_verify": true,
    "ca_cert_string": ""
  }
}
```
More than one address switches automatically to a Redis Cluster client.
</details>

<details>
<summary><code>pkg/config/mongodb.txt</code></summary>

```json
{ "mongo_string": "mongodb://user:password@127.0.0.1:27017/?authSource=admin" }
```
A full MongoDB connection URI - unlike the other stores, auth/TLS are
expected to be encoded in the URI itself rather than broken into separate
structured fields.
</details>

<details>
<summary><code>pkg/config/mail.txt</code></summary>

```json
{
  "mail_connections": [
    { "host": "smtp.example.com", "port": 587, "username": "bot@example.com", "password": "..." }
  ]
}
```
Multiple entries are load-balanced across for sending OTP emails.
</details>

<details>
<summary><code>pkg/config/groq_keys.txt</code> / <code>pkg/config/gemini_keys.txt</code></summary>

```json
{ "groq_keys": ["gsk_..."] }
```
```json
{ "gemini_keys": ["AIza..."] }
```
Multiple keys are round-robined across for every call.
</details>

The repo ships with a working dev config already embedded (see
`pkg/*_test.go` files that exercise it), so you can run the test suite
without setting anything up — but standing up your own instance means
re-encrypting your own blobs into these six files.

### Running the server

The entrypoint is `main.go` at the repo root:

```go
package main

import (
    "hidden-prompt-backend/pkg/core/deps"
    env "hidden-prompt-backend/pkg/environment"
)

func main() {
    env.SetDotEnv("./")
    deps.NewApp().Run()
}
```

`deps.NewApp()` builds the full dependency graph (config → clients →
database → services → handlers → middleware → router) as an `uber-go/fx`
app; `.Run()` (fx's own method) starts everything — including the
`fasthttp` server, which always listens on port `8080` (hardcoded in
`pkg/core/deps/di.go`, not configurable via `$PORT` or any other env var)
— and the DB-keepalive cron job, then blocks until it receives
`SIGINT`/`SIGTERM`, at which point it shuts down gracefully, draining
in-flight requests.

```bash
go run .
```

### Running tests

```bash
go build ./...
go vet ./...
go test ./...
```

Some test files (`pkg/clients/groq`, `pkg/clients/gemini`, `pkg/mail`,
`pkg/database`) are **live integration tests** — they make real calls
against Groq/Gemini/SMTP/your configured databases using the embedded dev
config. `pkg/utils`, `pkg/chat`, and `pkg/service` tests are fully mocked
and don't touch the network.

## Authentication

There's no session store or JWT — identity is a single encrypted token:

1. `POST /users/signup` or `POST /users/login` succeeds and the response
   carries an `X-Auth-Token` header: the user's email, AES-SIV-encrypted
   (`pkg/utils.EncryptString`).
2. The client stores that header value and sends it back as `X-Auth-Token`
   on every subsequent request to a protected endpoint.
3. The server decrypts it (`pkg/utils.DecryptString`) to recover the
   email. A missing or undecryptable token gets a `401` with the exact
   message `"Please LogIn or SignUp"`.

See [API.md](./API.md) for exactly which endpoints require this header.

A separate, smaller set of endpoints — the `/admin/*` group and
`POST /music` — require an `X-Admin-Key` header instead, checked against a
single static shared secret rather than a per-user token. See
[API.md](./API.md#authentication) for details.

## Rate limiting

Every handler is rate-limited via Redis (`go-redis/redis_rate`'s GCRA
algorithm — it smooths bursts rather than enforcing a hard fixed-window
cliff). Limits are tuned for a small, resource-constrained, free-tier
deployment and are documented per-endpoint in [API.md](./API.md). A
`429` response includes a `Retry-After` header (seconds). If Redis itself
is unavailable, rate limiting **fails open** (requests are allowed
through) rather than taking the whole API down over a limiter hiccup.

## Error handling conventions

Every error response is JSON:

```json
{
  "show_response_as_is": true,
  "error_message": "The email or password you entered is incorrect.",
  "status_code": 401
}
```

`show_response_as_is: true` means `error_message` is a safe, polished,
user-facing sentence you can display directly. `false` means the message
is a lower-level/internal error string — safe to log, not meant to be
shown verbatim to an end user (this only happens for unexpected `5xx`s and
raw request-body parse failures).

Every response — success or failure — also carries an `X-Time-Taken`
header (e.g. `12.4ms`) reporting server-side handling time for that
request.

## Observability

- `GET /health` reports MySQL/PostgreSQL/Redis/MongoDB connectivity
  individually (`200` if all healthy, `503` with a per-dependency error map
  otherwise).
  It is intentionally **not** rate-limited, since uptime monitors and load
  balancers poll it frequently.
- Panics anywhere in a handler are recovered by the outermost middleware,
  logged with a full stack trace via `log/slog`, and turned into a clean
  `{"panic": "..."}` `500` instead of crashing the process.

## License

MIT — see [LICENSE](./LICENSE).
