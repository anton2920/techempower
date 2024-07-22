package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var HelloWorld = []byte("Hello, world!\n")

func PlaintextHandler(w http.ResponseWriter, r *http.Request) {
	w.Write(HelloWorld)
}

type Fortune struct {
	ID      int
	Message string
}

var (
	DB   *pgxpool.Pool
	Tmpl *template.Template
)

func FortunesHandler(w http.ResponseWriter, r *http.Request) {
	rows, _ := DB.Query(context.Background(), "SELECT id, message FROM fortunes")
	fortunes, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Fortune])
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Failed to read fortunes from DB: %v", err)
		return
	}
	fortunes = append(fortunes, Fortune{Message: "Additional fortune added at request time."})

	sort.Slice(fortunes, func(i, j int) bool {
		return fortunes[i].Message < fortunes[j].Message
	})

	if err := Tmpl.ExecuteTemplate(w, "fortunes.tmpl", fortunes); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Failed to execute fortunes template: %v", err)
		return
	}
}

func main() {
	var err error
	DB, err = pgxpool.New(context.Background(), "postgres://postgres:pass@localhost:5432/techempower")
	if err != nil {
		log.Fatalf("Failed to connect to a database: %v", err)
	}
	Tmpl = template.Must(template.ParseFiles("fortunes.tmpl"))

	http.HandleFunc("/plaintext", PlaintextHandler)
	http.HandleFunc("/fortunes", FortunesHandler)

	log.Printf("Listening on 0.0.0.0:7073...")
	log.Fatal(http.ListenAndServe("0.0.0.0:7073", nil))
}
