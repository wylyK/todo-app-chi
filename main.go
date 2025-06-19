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
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/wylyK/todo-app-chi/todo"
)

type queriesWrapper struct {
	queries *todo.Queries
}

func (q queriesWrapper) getNotesEndpoint(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	select {
	case <-time.After(3 * time.Second):
	case <-ctx.Done():
		log.Println("Request cancelled")
		return
	}

	params := r.URL.Query()["page"]
	if len(params) != 1 {
		http.Error(w, "Must specify a single page number", http.StatusBadRequest)
	}

	offset, err := strconv.ParseInt(params[0], 10, 32)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	notes, err := q.queries.GetNotesFromDB(ctx, int32(offset))
	if err != nil {
		if errors.Is(err, context.Canceled) {
			log.Println("Query cancelled")
			return
		}
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
	}
}

func (q *queriesWrapper) getNotesByIdEndpoint(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	select {
	case <-time.After(3 * time.Second):
	case <-ctx.Done():
		log.Println("Request cancelled")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	note, err := q.queries.GetNoteByIdFromDB(ctx, id)
	if err != nil {
		switch {
		case errors.Is(err, context.Canceled):
			log.Println("Query cancelled")
		case errors.Is(err, sql.ErrNoRows):
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
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
	}
}

func (q *queriesWrapper) postNotesEndpoint(w http.ResponseWriter, r *http.Request) {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	select {
	case <-time.After(3 * time.Second):
	case <-ctx.Done():
		log.Println("Request cancelled")
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
	noteRequest.ID = newId
	noteId, err := q.queries.PostNoteToDB(ctx, noteRequest)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			log.Println("Query cancelled")
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	id_b, err := json.Marshal(noteId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = w.Write(id_b)
	if err != nil {
		log.Println("Cannot write to response writer: " + err.Error())
	}
}

//go:embed schema.sql
var cmd string

func main() {
	db, err := sql.Open("pgx", "postgres://willy:2015@localhost:5432/database.postgres?sslmode=disable")
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
