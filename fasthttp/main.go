package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sort"

	"github.com/valyala/fasthttp"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var HelloWorld = []byte("Hello, world!\n")

func PlaintextHandler(ctx *fasthttp.RequestCtx) {
	switch string(ctx.Path()) {
	case "/plaintext":
		ctx.Write(HelloWorld)
	case "/fortunes":
		FortunesHandler(ctx)
	}
}

type Fortune struct {
	ID      int
	Message string
}

var DB *pgxpool.Pool

func FortunesHandler(ctx *fasthttp.RequestCtx) {
	rows, _ := DB.Query(context.Background(), "SELECT id, message FROM fortunes")
	fortunes, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Fortune])
	if err != nil {
		ctx.SetStatusCode(http.StatusInternalServerError)
		fmt.Fprintf(ctx, "Failed to read fortunes from DB: %v", err)
		return
	}
	fortunes = append(fortunes, Fortune{ID: len(fortunes), Message: "Additional fortune added at request time."})

	sort.Slice(fortunes, func(i, j int) bool {
		return fortunes[i].Message < fortunes[j].Message
	})

	component := FortunesTempl(fortunes)
	component.Render(context.Background(), ctx)
}

func main() {
	var err error
	DB, err = pgxpool.New(context.Background(), "postgres://postgres:pass@localhost:5432/techempower")
	if err != nil {
		log.Fatalf("Failed to connect to a database: %v", err)
	}

	log.Printf("Listening on 0.0.0.0:7073...")
	log.Fatal(fasthttp.ListenAndServe("0.0.0.0:7073", PlaintextHandler))
}
