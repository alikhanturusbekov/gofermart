package entity

import (
	"github.com/google/uuid"
	"time"
)

type Withdrawal struct {
	ID          uuid.UUID `json:"-"`
	UserID      uuid.UUID `json:"-"`
	OrderNumber string    `json:"order"`
	Sum         *float64  `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
}
