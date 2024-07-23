package model

import "github.com/anton2920/techempower/clean/internal/entity"

type Fortune struct {
	ID      int
	Message string
}

func (f *Fortune) ToEntity() entity.Fortune {
	return entity.Fortune{
		ID:      f.ID,
		Message: f.Message,
	}
}
