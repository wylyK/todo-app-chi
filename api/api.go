package api

import "github.com/google/uuid"

type Note struct {
	Id      uuid.UUID `json:"id"`
	Title   string    `json:"title"`
	Content string    `json:"content"`
}

type NoteRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}
