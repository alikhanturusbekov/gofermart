package entity

import (
	"github.com/google/uuid"
	"time"
)

type OrderStatus string

const (
	StatusNew        OrderStatus = "NEW"
	StatusRegistered OrderStatus = "REGISTERED"
	StatusProcessing OrderStatus = "PROCESSING"
	StatusProcessed  OrderStatus = "PROCESSED"
	StatusInvalid    OrderStatus = "INVALID"
)

type Order struct {
	ID         uuid.UUID   `json:"-"`
	UserID     uuid.UUID   `json:"-"`
	Number     string      `json:"number"`
	Status     OrderStatus `json:"status"`
	Accrual    *float64    `json:"accrual,omitempty"`
	UploadedAt time.Time   `json:"uploaded_at"`
}

type OrderProcessTask struct {
	Number string `json:"order_id"`
}
