# Hidden Prompt

A guessing game with a twist: instead of asking an AI a question and
reading its answer, you see only the AI's terse, one-line **answer** and
have to reverse-engineer the *question that produced it*.

> The AI says **"42"**. What did the player ask it?
> (`What is forty two in numeric`)

Guesses are scored for *intent*, not exact wording — a blend of embedding
cosine similarity and an LLM's own judgment of how close the guess's
meaning is to the real hidden prompt. A losing guess earns a hint,
calibrated to the player's skill and their guess history so far, without
ever revealing the answer outright.

## Repository layout

This repo holds two independent projects, each deployed on its own:

| Folder | What it is | Docs |
|---|---|---|
| [`hidden-prompt-backend/`](./hidden-prompt-backend) | Go API server: puzzle generation, guess scoring, hints, accounts | [README](./hidden-prompt-backend/README.md), [API reference](./hidden-prompt-backend/API.md) |
| [`hidden-prompt-frontend/`](./hidden-prompt-frontend) | Next.js web client (synthwave-themed) | [README](./hidden-prompt-frontend/README.md) |

Each folder has its own setup instructions, environment variables, and
deploy notes — start there for anything project-specific. This top-level
README is just an orientation point, not a substitute for those.

## License

MIT — see [LICENSE](./LICENSE). Each subproject also carries its own copy.
