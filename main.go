package main

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/wylyK/todo-app-chi/todo"
	_ "modernc.org/sqlite"
)

type queriesWrapper struct {
	queries *todo.Queries
}

func (q queriesWrapper) getNotesEndpoint(w http.ResponseWriter, r *http.Request) {
	notes, err := q.queries.GetNotesFromDB(context.TODO())
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
		log.Println("Cannot write to response writer: " + err.Error())
		return
	}
}

func (q *queriesWrapper) getNotesByIdEndpoint(w http.ResponseWriter, r *http.Request) {
	id := []byte(chi.URLParam(r, "id"))
	note, err := q.queries.GetNoteByIdFromDB(context.TODO(), id)
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
		log.Println("Cannot write to response writer: " + err.Error())
		return
	}
}

func (q *queriesWrapper) postNotesEndpoint(w http.ResponseWriter, r *http.Request) {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var noteRequest todo.PostNoteToDBParams
	err = json.Unmarshal(b, &noteRequest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(os.Stdout, "%+v, %+v\n", noteRequest.Title, noteRequest.Content)

	newId, err := uuid.NewRandom()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	noteRequest.ID = []byte(newId.String())

	noteId, err := q.queries.PostNoteToDB(context.TODO(), noteRequest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_, err = w.Write(noteId)
	if err != nil {
		log.Println("Cannot write to response writer: " + err.Error())
	}
}

//go:embed schema.sql
var cmd string

func main() {
	db, err := sql.Open("sqlite", "database.sqlite")
	if err != nil {
		log.Fatal("failed to open file: " + err.Error())
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connected!")

	_, err = db.Exec(cmd)
	if err != nil {
		log.Fatal(err)
	}

	q := &queriesWrapper{queries: todo.New(db)}

	r := chi.NewRouter()
	r.Get("/notes", q.getNotesEndpoint)
	r.Get("/notes/{id}", q.getNotesByIdEndpoint)
	r.Post("/notes", q.postNotesEndpoint)

	err = http.ListenAndServe(":8080", r)
	if err != nil {
		log.Fatal(err)
	}
}
