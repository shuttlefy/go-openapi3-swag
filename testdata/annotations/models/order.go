package models

import "time"

// OrderStatus 订单状态（string enum）。
type OrderStatus string

const (
	OrderStatusPlaced    OrderStatus = "placed"
	OrderStatusApproved  OrderStatus = "approved"
	OrderStatusDelivered OrderStatus = "delivered"
)

// Order 订单。引用 OrderStatus（enum），含 time.Time 字段。
type Order struct {
	ID        int64       `json:"id"`
	PetID     int64       `json:"pet_id"`
	Quantity  int         `json:"quantity"   minimum:"1" maximum:"100"`
	ShipDate  time.Time   `json:"ship_date,omitempty"`
	Status    OrderStatus `json:"status"     enums:"placed,approved,delivered"`
	Complete  bool        `json:"complete"`
	CreatedAt time.Time   `json:"created_at" readonly:"true"`
}

// CreateOrderRequest 创建订单请求体。
type CreateOrderRequest struct {
	PetID    int64 `json:"pet_id"   validate:"required"`
	Quantity int   `json:"quantity" validate:"required" minimum:"1" maximum:"100"`
}
