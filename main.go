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

func getNotesEndpoint(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		param, hasId := r.URL.Query()["id"]
		if !hasId {
			cmd := "SELECT id, title, content FROM notes"
			entry, err := db.Query(cmd)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			notes := []api.Note{}

			for entry.Next() {
				var currNote api.Note
				err = entry.Scan(&currNote.Id, &currNote.Title, &currNote.Content)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				notes = append(notes, currNote)
			}

			b, err := json.Marshal(notes)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Write(b)
			return
		}

		id := param[0]
		cmd := "SELECT id, title, content FROM notes WHERE id=?"
		entry := db.QueryRow(cmd, id)

		var note api.Note

		err := entry.Scan(&note.Id, &note.Title, &note.Content)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}

			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
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
		var note api.NoteRequest

		b, err := io.ReadAll(r.Body)
		if err != nil {
			fmt.Fprint(w, err)
			return
		}

		if err = json.Unmarshal(b, &note); err != nil {
			fmt.Fprint(w, err)
			return
		}

		fmt.Fprintf(os.Stdout, "%+v", note)

		cmd := "INSERT INTO notes (id, title, content) VALUES(?, ?, ?) RETURNING id"
		newKey, err := uuid.NewRandom()
		if err != nil {
			fmt.Fprint(w, err)
			return
		}

		var noteId uuid.UUID

		entry := db.QueryRow(cmd, newKey, note.Title, note.Content)
		err = entry.Scan(&noteId)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write([]byte(noteId.String()))
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
	r.Post("/notes", postNotesEndpoint(db))

	err = http.ListenAndServe(":8080", r)
	if err != nil {
		log.Fatal(err)
	}
}
