package entity

import (
	"github.com/google/uuid"
	"time"
)

type OrderStatus string

const (
	StatusNew        OrderStatus = "NEW"
	StatusProcessing OrderStatus = "PROCESSING"
	StatusProcessed  OrderStatus = "PROCESSED"
	StatusInvalid    OrderStatus = "INVALID"
)

type Order struct {
	ID         uuid.UUID   `json:"id"`
	UserID     uuid.UUID   `json:"user_id"`
	Number     string      `json:"order_id"`
	Status     OrderStatus `json:"status"`
	Accrual    *float64    `json:"accrual;omitempty"`
	UploadedAt time.Time   `json:"uploaded_at"`
}
