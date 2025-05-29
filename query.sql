-- name: GetNoteByIdFromDB :one
SELECT * FROM notes
WHERE id = ?;

-- name: GetNotesFromDB :many
SELECT * FROM notes;

-- name: PostNoteToDB :one
INSERT INTO notes (
    id, title, content
) VALUES (
    ?, ?, ?
) RETURNING id;
