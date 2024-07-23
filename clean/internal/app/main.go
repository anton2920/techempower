package app

import (
	"context"
	"log"

	"github.com/anton2920/techempower/clean/internal/handler"
	"github.com/anton2920/techempower/clean/internal/repository/postgres"
	"github.com/anton2920/techempower/clean/internal/server"
	"github.com/anton2920/techempower/clean/internal/service"
)

func Run() {
	fortunesRepo, err := postgres.NewFortunesRepository(context.Background(), "postgres://postgres:pass@localhost:5432/techempower")
	if err != nil {
		log.Fatalf("Failed to create new fortunes postgresql repository: %v", err)
	}
	fortunesService := service.NewFortunesService(fortunesRepo)

	handler, err := handler.New(fortunesService)
	if err != nil {
		log.Fatalf("Failed to create new handler: %v", err)
	}
	server := server.New("0.0.0.0:7073", handler)

	log.Printf("Listening on 0.0.0.0:7073...")
	log.Fatal(server.Run())
}
