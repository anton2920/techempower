package repository

import (
	"context"

	"github.com/anton2920/techempower/clean/internal/entity"
)

type FortunesRepository interface {
	GetAll(context.Context) ([]entity.Fortune, error)
}
