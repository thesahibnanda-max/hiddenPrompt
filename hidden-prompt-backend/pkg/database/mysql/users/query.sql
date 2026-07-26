-- name: CreateUser :exec
INSERT INTO users (
    user_email,
    password_hash,
    is_verified,
    metadata
) VALUES (
    ?, ?, ?, ?
);

-- name: GetUserByEmail :one
SELECT *
FROM users
WHERE user_email = ?
LIMIT 1;

-- name: UpdateUserVerificationToTrue :exec
UPDATE users
SET
    is_verified = TRUE
WHERE user_email = ?;

-- name: DeleteUserByEmail :exec
DELETE FROM users
WHERE user_email = ?;