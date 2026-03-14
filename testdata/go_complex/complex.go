package complex

import (
	"context"
	"net/http"
	"time"
)

// --- Embedded structs ---

type BaseModel struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Product struct {
	BaseModel
	Name        string            `json:"name" binding:"required"`
	Price       float64           `json:"price"`
	Metadata    map[string]any    `json:"metadata,omitempty"`
	Variants    []*ProductVariant `json:"variants,omitempty"`
	CategoryIDs []int64           `json:"category_ids"`
}

type ProductVariant struct {
	SKU   string  `json:"sku"`
	Color *string `json:"color"`
	Size  string  `json:"size"`
}

// --- Enum types ---

// OrderStatus is an enum-like type.
type OrderStatus string

const (
	// OrderStatusPending means the order is awaiting payment.
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusProcessing OrderStatus = "processing"
	OrderStatusShipped    OrderStatus = "shipped"
	OrderStatusDelivered  OrderStatus = "delivered"
	// OrderStatusCancelled means the order was cancelled.
	OrderStatusCancelled OrderStatus = "cancelled"
)

type Priority int

const (
	PriorityLow Priority = iota
	PriorityMedium
	PriorityHigh
	PriorityCritical
)

// --- Type block declaration ---

type (

	// Order represents a purchase order.
	Order struct {
		BaseModel
		// UserID is the buyer.
		UserID int64               `json:"user_id"`
		Status OrderStatus         `json:"status"`
		Items  []OrderItem         `json:"items"`
		ShipTo *Address            `json:"ship_to,omitempty"`
		Extra  map[string][]string `json:"extra,omitempty"`
	}

	OrderItem struct {
		ProductID int64   `json:"product_id"`
		Quantity  int     `json:"quantity"`
		UnitPrice float64 `json:"unit_price"`
	}

	Address struct {
		Line1   string `json:"line1"`
		Line2   string `json:"line2,omitempty"`
		City    string `json:"city"`
		Country string `json:"country"`
		Zip     string `json:"zip"`
	}
)

// --- Empty struct ---

type Empty struct{}

// --- Complex field types ---

type ComplexFields struct {
	Tags      map[string]interface{} `json:"tags"`
	Matrix    [][]int                `json:"matrix"`
	Nested    *[]*Product            `json:"nested"`
	Callback  func()
	ChanField chan int
}

// --- Functions with complex signatures ---

// ListProducts godoc
// @Summary     List all products
// @Description Retrieve paginated product list with filters
// @Tags        products
// @Param       offset query int false "Pagination offset"
// @Param       limit  query int false "Page size"
// @Param       q      query string false "Search keyword"
// @Success     200 {array} Product
// @Failure     500 {object} ErrorResp
// @Router      /products [get]
func ListProducts(ctx context.Context, offset, limit int, q string) ([]*Product, int64, error) {
	return nil, 0, nil
}

func NoDocFunction() {}

// VariadicFunc has variadic params.
func VariadicFunc(prefix string, ids ...int64) error {
	return nil
}

// NamedReturns uses named return values.
func NamedReturns(id int) (product *Product, found bool, err error) {
	return nil, false, nil
}

// SelectorParams uses types from external packages.
func SelectorParams(w http.ResponseWriter, r *http.Request) {
}

type OrderService struct{}

// CreateOrder is a method on OrderService.
// @Summary     Create order
// @Description Place a new order
// @Tags        orders
// @Param       body body Order true "Order payload"
// @Success     201 {object} Order
// @Failure     400 {object} ErrorResp
// @Router      /orders [post]
func (s *OrderService) CreateOrder(ctx context.Context, order *Order) (*Order, error) {
	return nil, nil
}

// CancelOrder is a method with a value receiver.
// @Summary Cancel order
// @Router  /orders/{id}/cancel [post]
func (s OrderService) CancelOrder(ctx context.Context, id int64) error {
	return nil
}

type ErrorResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
