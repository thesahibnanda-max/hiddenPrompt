CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS puzzles (
    puzzle_id VARCHAR(255) PRIMARY KEY,

    user_email VARCHAR(255) NOT NULL,

    hidden_prompt TEXT NOT NULL,

    canonical_answer TEXT NOT NULL,

    hidden_prompt_embedding VECTOR(768) NOT NULL,

    messages JSONB NOT NULL,

    metadata JSONB NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_puzzles_user_email
ON puzzles(user_email);

CREATE INDEX IF NOT EXISTS idx_puzzles_created_at
ON puzzles(created_at);

-- Matches GetPuzzlesByUser's exact WHERE + ORDER BY shape, so Postgres can
-- satisfy the whole query with a single index scan instead of an index
-- scan on user_email followed by a separate in-memory sort.
CREATE INDEX IF NOT EXISTS idx_puzzles_user_email_created_at
ON puzzles(user_email, created_at DESC);