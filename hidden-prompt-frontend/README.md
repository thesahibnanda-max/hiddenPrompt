# Hidden Prompt — Frontend

A synthwave-themed web client for [Hidden Prompt](../hidden-prompt-backend),
the reverse-prompt-guessing game. Next.js 14 (App Router) + TypeScript +
Tailwind + shadcn/ui-style components + Framer Motion + Zustand + Howler.js.

## Local development

1. Install dependencies:
   ```bash
   npm install
   ```
2. Copy the env file and point it at your backend:
   ```bash
   cp .env.example .env.local
   # edit NEXT_PUBLIC_API_BASE_URL if not using the default
   ```
3. Run the backend (from `../hidden-prompt-backend`):
   ```bash
   go run .
   ```
   This uses `dev.env` and the checked-in dev config, and listens on port
   `8080` by default — matching `.env.example`'s default out of the box.
4. Run the frontend dev server:
   ```bash
   npm run dev
   ```

## Configuring the API base URL

The entire "point this at a different backend" story is one environment
variable: `NEXT_PUBLIC_API_BASE_URL`, consumed in exactly one place
(`src/lib/api/client.ts`). No other code changes are needed.

**Important:** `NEXT_PUBLIC_*` variables are inlined into the JavaScript
bundle at **build time**, not read at runtime. If you change this value on
an already-deployed instance, you need to rebuild/redeploy — flipping the
env var alone on a running server won't do anything.

## Production build

```bash
npm run build
npm run start
```

Requires Node 18+. Works behind any reverse proxy (Nginx, Caddy, etc.) on
whatever port you configure `next start` for (`-p <port>`, or the `PORT` env
var, which Next respects natively).

## Deploying to Vercel

This project lives in its own folder (`hidden-prompt-frontend/`), a sibling
of `hidden-prompt-backend/` — not nested inside it — so importing it as a
Vercel project needs no root-directory override. Steps:

1. Import this folder as a new Vercel project.
2. Set `NEXT_PUBLIC_API_BASE_URL` in Project Settings → Environment
   Variables to your deployed backend's public URL.
3. Deploy. Changing the env var later requires a redeploy (see caveat above).

No backend/CORS configuration changes are required for any deploy target —
the backend already allows all origins.

## Audio placeholders

`public/audio/1.mp3` and `public/audio/2.mp3` are minimal generated silent
placeholder tracks (there's no `ffmpeg`/`lame` in this environment to encode
real audio). They're sequenced back-to-back as a two-track playlist by
`src/lib/audio/howler-manager.ts`. Replace them with real synthwave loops
using the **same filenames** and everything else — muting, persistence,
autoplay-policy handling — keeps working unchanged.

## Security

Pinned to `next@14.2.35` (latest 14.x patch), which resolves the critical
Server Actions DoS advisory present in earlier 14.2.x releases. `npm audit`
still reports several high/moderate advisories against the `next` package
range — these are in `next/image` remote-pattern handling, Server Actions,
Next.js Middleware, and i18n routing, none of which this app uses (no
`next/image`, no Server Actions, no `middleware.ts`, no i18n config). Treat
as an accepted risk for this build, not a blind pass — re-run `npm audit`
and consider a Next 15/16 upgrade before a real production launch.

## Known, intentional simplifications

- **Puzzle-card win badges** are not derived from a live status fetch (`GET
  /puzzle` doesn't return win state, and fetching detail for every card would
  be an N+1 request storm against a 30/min rate limit). Badges only populate
  once a puzzle has actually been opened this session — see
  `src/lib/utils/puzzle-outcomes.ts`.
- **Route protection is client-side only** (no Next.js middleware) since the
  auth token is opaque and stored in `localStorage`, which edge middleware
  can't meaningfully read.
- The backend always listens on port `8080` (hardcoded in
  `pkg/core/deps/di.go`, not configurable via `$PORT` or any other env
  var) — keep that in mind when deploying it, since there's no way to ask
  it to bind elsewhere without a code change.
