package postgres

import (
	"context"
	"fmt"

	"github.com/anton2920/techempower/clean/internal/entity"
	"github.com/anton2920/techempower/clean/internal/model"
	"github.com/anton2920/techempower/clean/internal/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type fortunesRepository struct {
	db *pgxpool.Pool
}

func NewFortunesRepository(ctx context.Context, dsn string) (repository.FortunesRepository, error) {
	var r fortunesRepository
	var err error

	r.db, err = pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create new pool of DB connections: %w", err)
	}

	return &r, nil
}

func (r *fortunesRepository) GetAll(ctx context.Context) ([]entity.Fortune, error) {
	rows, _ := r.db.Query(ctx, "SELECT id, message FROM fortunes")
	models, err := pgx.CollectRows(rows, pgx.RowToStructByPos[model.Fortune])
	if err != nil {
		return nil, fmt.Errorf("failed to get fortune models from DB: %w", err)
	}

	entities := make([]entity.Fortune, len(models))
	for i := 0; i < len(models); i++ {
		entities[i] = models[i].ToEntity()
	}

	return entities, nil
}
