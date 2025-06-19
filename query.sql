-- name: GetNoteByIdFromDB :one
SELECT * FROM notes
WHERE id = $1 LIMIT 1;

-- name: GetNotesFromDB :many
SELECT * FROM notes
ORDER BY created_at DESC
LIMIT 5 OFFSET $1;

-- name: PostNoteToDB :one
INSERT INTO notes (
    id, title, content
) VALUES (
    $1, $2, $3
) RETURNING id;
