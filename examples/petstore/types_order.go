package main

import "time"

// ── Order domain types ────────────────────────────────────────────────────────
//
// These types live in a separate file from main.go to demonstrate that swag3
// can resolve cross-file struct references when scanning a whole directory.
// Annotations in main.go (handler methods) reference CreateOrderRequest and
// Order, whose fields in turn reference Address and OrderItem — all resolved
// from this file at generation time.

// Address is a physical mailing address.
// It is used both inline (ShippingAddress) and as a nullable pointer
// (BillingAddress) inside CreateOrderRequest, producing two distinct schema
// patterns in the generated spec:
//
//	shipping_address: { $ref: Address }          – required inline value
//	billing_address:  { $ref: Address, nullable } – optional pointer
type Address struct {
	Street  string `json:"street"            binding:"required" example:"123 Main St"`
	City    string `json:"city"              binding:"required" example:"San Francisco"`
	Country string `json:"country"           binding:"required" example:"US"`
	ZipCode string `json:"zip_code,omitempty"                  example:"94105"`
}

// OrderItem is a single line item inside an order.
// An order's Items field is []OrderItem, demonstrating that swag3 generates
// an array schema whose items carry their own required/validation metadata.
type OrderItem struct {
	PetID    int64  `json:"pet_id"            binding:"required" example:"1"`
	Quantity int32  `json:"quantity"          binding:"required" example:"2"`
	Note     string `json:"note,omitempty"                      example:"gift wrap please"`
}

// CreateOrderRequest is a deliberately complex request body that exercises
// several schema-generation features simultaneously:
//
//   - Deep nesting     ShippingAddress Address   (inline struct value)
//   - Nullable pointer BillingAddress  *Address  (same struct, optional)
//   - Array of structs Items           []OrderItem
//   - stdlib time.Time PlacedAt        time.Time → OpenAPI string/date-time
type CreateOrderRequest struct {
	CustomerName    string      `json:"customer_name"    binding:"required" example:"Alice"`
	ShippingAddress Address     `json:"shipping_address" binding:"required"`
	BillingAddress  *Address    `json:"billing_address,omitempty"`
	Items           []OrderItem `json:"items"            binding:"required"`
	Note            string      `json:"note,omitempty"                     example:"please deliver by Friday"`
	// PlacedAt lets the caller specify when the order was placed (e.g. backfill).
	// Mapped to OpenAPI string (format: date-time) via the time.Time mapping rule.
	PlacedAt time.Time `json:"placed_at"`
}

// Order is the persisted representation returned by the API.
// All nested types are shared with CreateOrderRequest, so the generated
// components/schemas section reuses the same $ref entries for Address and
// OrderItem rather than inlining them.
type Order struct {
	ID              int64       `json:"id"                           example:"1"`
	CustomerName    string      `json:"customer_name"                example:"Alice"`
	ShippingAddress Address     `json:"shipping_address"`
	BillingAddress  *Address    `json:"billing_address,omitempty"`
	Items           []OrderItem `json:"items"`
	// Status uses the OrderStatus alias type so the enum is centralised in
	// components/schemas/OrderStatus and referenced here via $ref.
	Status    OrderStatus `json:"status"                       example:"pending"`
	Note      string      `json:"note,omitempty"               example:"please deliver by Friday"`
	PlacedAt  time.Time   `json:"placed_at"`
	UpdatedAt *time.Time  `json:"updated_at,omitempty"`
}
