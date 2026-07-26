-- name: CreatePuzzle :one
INSERT INTO puzzles (
    puzzle_id,
    user_email,
    hidden_prompt,
    canonical_answer,
    hidden_prompt_embedding,
    messages,
    metadata
)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7
)
RETURNING *;

-- name: GetPuzzleByID :one
SELECT *
FROM puzzles
WHERE puzzle_id = $1;


-- name: GetPuzzlesByUser :many
-- Deliberately narrow: still excludes hidden_prompt_embedding and messages,
-- the genuinely expensive columns (a 768-dim vector plus a growing JSONB
-- blob, per row) - full puzzle data for those still goes through
-- GetPuzzleByID instead. hidden_prompt itself is a plain short text column
-- (comparable cost to canonical_answer, already selected below) and is
-- included so puzzle generation can build a per-user "don't repeat these
-- questions" list without a second query.
SELECT puzzle_id, canonical_answer, hidden_prompt, metadata, created_at, updated_at
FROM puzzles
WHERE user_email = $1
ORDER BY created_at DESC;

-- name: UpdatePuzzleMessages :one
-- Optimistic concurrency: the caller must pass the updated_at it last read
-- (current_updated_at). If another write has landed in the meantime, this
-- matches zero rows instead of blindly overwriting the newer data.
UPDATE puzzles
SET
    messages = $2,
    updated_at = NOW()
WHERE puzzle_id = $1 AND updated_at = $3
RETURNING *;

-- name: UpdatePuzzleMetadata :one
-- Optimistic concurrency: see UpdatePuzzleMessages above.
UPDATE puzzles
SET
    metadata = $2,
    updated_at = NOW()
WHERE puzzle_id = $1 AND updated_at = $3
RETURNING *;

-- name: DeletePuzzlesByUser :exec
DELETE FROM puzzles
WHERE user_email = $1;