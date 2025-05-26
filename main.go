package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/wylyK/todo-app-chi/api"
	_ "modernc.org/sqlite"
)

func getNotesFromDB(db *sql.DB) ([]api.Note, error) {
	notes := []api.Note{}
	cmd := "SELECT id, title, content FROM notes"
	entry, err := db.Query(cmd)
	if err != nil {
		return notes, err
	}

	for entry.Next() {
		var currNote api.Note
		err = entry.Scan(&currNote.Id, &currNote.Title, &currNote.Content)
		if err != nil {
			return notes, err
		}

		notes = append(notes, currNote)
	}

	return notes, nil
}

func getNoteByIdFromDB(db *sql.DB, id string) (api.Note, error) {
	var note api.Note
	cmd := "SELECT id, title, content FROM notes WHERE id=?"
	entry := db.QueryRow(cmd, id)

	err := entry.Scan(&note.Id, &note.Title, &note.Content)
	if err != nil {
		return note, err
	}

	return note, nil
}

func postNoteToDB(db *sql.DB, noteRequest api.NoteRequest) (uuid.UUID, error) {
	var noteId uuid.UUID
	cmd := "INSERT INTO notes (id, title, content) VALUES(?, ?, ?) RETURNING id"
	newKey, err := uuid.NewRandom()
	if err != nil {
		return noteId, err
	}

	entry := db.QueryRow(cmd, newKey, noteRequest.Title, noteRequest.Content)
	err = entry.Scan(&noteId)
	if err != nil {
		return noteId, err
	}

	return noteId, nil
}

func getNotesEndpoint(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		notes, err := getNotesFromDB(db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		b, err := json.Marshal(notes)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = w.Write(b)
		if err != nil {
			log.Print("Cannot write to response writer: " + err.Error())
			return
		}
	}
}

func getNotesByIdEndpoint(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		note, err := getNoteByIdFromDB(db, id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}

			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		b, err := json.Marshal(note)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = w.Write(b)
		if err != nil {
			log.Print("Cannot write to response writer: " + err.Error())
			return
		}
	}
}

func postNotesEndpoint(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var noteRequest api.NoteRequest

		b, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err = json.Unmarshal(b, &noteRequest); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(os.Stdout, "%+v", noteRequest)

		noteId, err := postNoteToDB(db, noteRequest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = w.Write([]byte(noteId.String()))
		if err != nil {
			log.Print("Cannot write to response writer: " + err.Error())
			return
		}
		w.WriteHeader(http.StatusCreated)
	}
}

func main() {
	r := chi.NewRouter()

	db, err := sql.Open("sqlite", "database.sqlite")
	if err != nil {
		log.Fatal("failed to open file: " + err.Error())
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connected!")

	cmd := `CREATE TABLE IF NOT EXISTS notes (
    id blob,
    title text,
	content text
	);`

	_, err = db.Exec(cmd)
	if err != nil {
		log.Fatal(err)
	}

	r.Get("/notes", getNotesEndpoint(db))
	r.Get("/notes/{id}", getNotesByIdEndpoint(db))
	r.Post("/notes", postNotesEndpoint(db))

	err = http.ListenAndServe(":8080", r)
	if err != nil {
		log.Fatal(err)
	}
}
