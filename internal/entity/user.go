package entity

import (
	"github.com/google/uuid"
	"time"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	Login     string    `json:"login"`
	Password  string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}
