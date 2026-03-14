package petstore

import "time"

// swaggerInfo holds the global API annotations.
//
// @title Pet Store API
// @version 1.0.0
// @description A sample Pet Store API demonstrating swag3 annotation support.
// @host localhost:8088
// @BasePath /api/v1
// @tag pets "Operations about pets"
// @tag orders "Order management"
// @tag categories "Category tree"
// @securityDefinitions.apikey ApiKeyAuth
// @securityDefinitions.apikey.in header
// @securityDefinitions.apikey.name X-API-Key
func swaggerInfo() {}

// ── Named-type enums ─────────────────────────────────────────────────────────
//
// These type aliases with const blocks exercise swag3's alias + enum feature:
// each alias produces a named schema in components/schemas and the const
// values are collected as the enum array on that schema.

// PetStatus is the lifecycle state of a pet.
type PetStatus string

const (
	// PetStatusAvailable means the pet is visible and can be adopted.
	PetStatusAvailable PetStatus = "available"
	// PetStatusPending means an adoption application is in progress.
	PetStatusPending PetStatus = "pending"
	// PetStatusSold means the pet has been adopted and is no longer listed.
	PetStatusSold PetStatus = "sold"
)

// EventType describes the kind of lifecycle event for a pet.
type EventType string

const (
	// EventTypeCreated is emitted when a pet record is first inserted.
	EventTypeCreated EventType = "created"
	// EventTypeUpdated is emitted when any field on a pet is changed.
	EventTypeUpdated EventType = "updated"
	// EventTypeViewed is emitted when a pet detail page is fetched.
	EventTypeViewed EventType = "viewed"
	// EventTypeDeleted is emitted when a pet record is removed.
	EventTypeDeleted EventType = "deleted"
)

// OrderStatus is the fulfilment state of an order.
type OrderStatus string

const (
	// OrderStatusPending means the order has been placed but not yet paid.
	OrderStatusPending OrderStatus = "pending"
	// OrderStatusPaid means payment has been confirmed.
	OrderStatusPaid OrderStatus = "paid"
	// OrderStatusShipped means the order is in transit.
	OrderStatusShipped OrderStatus = "shipped"
	// OrderStatusDelivered means the order has been received by the customer.
	OrderStatusDelivered OrderStatus = "delivered"
	// OrderStatusCancelled means the order was cancelled before shipment.
	OrderStatusCancelled OrderStatus = "cancelled"
)

// PetTag is a free-form label for filtering pets.
type PetTag string

// ── Numeric-type enums ────────────────────────────────────────────────────────

// Priority is the scheduling priority of a task, ascending from lowest to highest.
// Values are derived from iota+1 so that zero is never a valid priority.
type Priority int

const (
	// PriorityLow is the default, background-level priority.
	PriorityLow Priority = iota + 1
	// PriorityNormal is the standard interactive priority.
	PriorityNormal
	// PriorityHigh is for time-sensitive operations.
	PriorityHigh
	// PriorityCritical is reserved for emergencies and alerts.
	PriorityCritical
)

// SortOrder controls the direction of a sort operation.
type SortOrder int8

const (
	// SortOrderDesc sorts results from largest to smallest.
	SortOrderDesc SortOrder = -1
	// SortOrderNone means no ordering guarantee.
	SortOrderNone SortOrder = 0
	// SortOrderAsc sorts results from smallest to largest.
	SortOrderAsc SortOrder = 1
)

// PageSize is a predefined page size for list endpoints.
type PageSize int32

const (
	// PageSizeSmall returns 10 items per page.
	PageSizeSmall PageSize = 10
	// PageSizeMedium returns 20 items per page.
	PageSizeMedium PageSize = 20
	// PageSizeLarge returns 50 items per page.
	PageSizeLarge PageSize = 50
	// PageSizeMax returns 100 items per page; the hard server-side cap.
	PageSizeMax PageSize = 100
)

// ── Top-level types ───────────────────────────────────────────────────────────

// BaseResponse is the universal response envelope for every API endpoint.
// Successful responses use code=0; application errors use a non-zero code.
// The Data field is overridden per endpoint via composite annotations, e.g.:
//
//	@Success 200 {object} BaseResponse{data=Pet}
//	@Success 200 {object} BaseResponse{data=PagedList}
type BaseResponse struct {
	Code    int         `json:"code"    example:"0"`
	Message string      `json:"message" example:"ok"`
	Data    interface{} `json:"data,omitempty"`
}

// PagedList is the pagination payload carried inside BaseResponse.Data.
// List holds the concrete item slice for the current page.
type PagedList struct {
	Total int64       `json:"total" example:"42"`
	List  interface{} `json:"list"`
}

// Pet represents a pet in the store.
// Demonstrates stdlib time.Time fields mapped to OpenAPI string/date-time.
type Pet struct {
	ID        int64      `json:"id"               example:"1"`
	Name      string     `json:"name"             binding:"required" example:"Buddy"`
	Status    PetStatus  `json:"status"           example:"available"`
	Tags      []PetTag   `json:"tags,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

// CreatePetRequest is the request body for creating a pet.
type CreatePetRequest struct {
	Name   string    `json:"name"   binding:"required" example:"Buddy"`
	Status PetStatus `json:"status" binding:"required" example:"available"`
	Tags   []PetTag  `json:"tags,omitempty"`
}

// PetEvent records a lifecycle event for a pet.
// Demonstrates third-party / stdlib struct field usage:
// - time.Time → OpenAPI string (format: date-time)
// - map[string]string → OpenAPI object with additionalProperties
type PetEvent struct {
	ID         int64             `json:"id"               example:"1"`
	PetID      int64             `json:"pet_id"           example:"1"`
	EventType  EventType         `json:"event_type"       example:"created"`
	Detail     string            `json:"detail,omitempty" example:"pet was created via API"`
	OccurredAt time.Time         `json:"occurred_at"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// UploadResult carries metadata about a successfully uploaded file.
// Returned in BaseResponse.Data after a multipart upload.
type UploadResult struct {
	Filename    string `json:"filename"     example:"avatar.jpg"`           // original filename from the client
	Size        int64  `json:"size"         example:"204800"`               // file size in bytes
	ContentType string `json:"content_type" example:"image/jpeg"`           // detected MIME type
	URL         string `json:"url"          example:"/api/v1/pets/1/avatar"` // relative download URL
}

// ── Order types ───────────────────────────────────────────────────────────────
// These types live in a separate file (types_order.go) in examples/petstore to
// demonstrate that swag3 resolves struct references across files in the same
// package. They are duplicated here only to make the testdata self-contained.

// Address is a physical mailing address embedded in orders.
type Address struct {
	Street  string `json:"street"            binding:"required" example:"123 Main St"`
	City    string `json:"city"              binding:"required" example:"San Francisco"`
	Country string `json:"country"           binding:"required" example:"US"`
	ZipCode string `json:"zip_code,omitempty"                  example:"94105"`
}

// OrderItem is a single line item inside an order.
type OrderItem struct {
	PetID    int64  `json:"pet_id"   binding:"required" example:"1"`
	Quantity int32  `json:"quantity" binding:"required" example:"2"`
	Note     string `json:"note,omitempty"              example:"gift wrap please"`
}

// CreateOrderRequest demonstrates a deeply nested request body:
//   - ShippingAddress Address    inline struct value (required)
//   - BillingAddress  *Address   same struct, nullable pointer
//   - Items           []OrderItem array of structs
//   - PlacedAt        time.Time  stdlib → string/date-time
type CreateOrderRequest struct {
	CustomerName    string      `json:"customer_name"    binding:"required" example:"Alice"`
	ShippingAddress Address     `json:"shipping_address" binding:"required"`
	BillingAddress  *Address    `json:"billing_address,omitempty"`
	Items           []OrderItem `json:"items"            binding:"required"`
	Note            string      `json:"note,omitempty"                     example:"please deliver by Friday"`
	PlacedAt        time.Time   `json:"placed_at"`
}

// Order is the persisted order returned by the API.
type Order struct {
	ID              int64       `json:"id"                           example:"1"`
	CustomerName    string      `json:"customer_name"                example:"Alice"`
	ShippingAddress Address     `json:"shipping_address"`
	BillingAddress  *Address    `json:"billing_address,omitempty"`
	Items           []OrderItem `json:"items"`
	Status          OrderStatus `json:"status"                       example:"pending"`
	Note            string      `json:"note,omitempty"               example:"please deliver by Friday"`
	PlacedAt        time.Time   `json:"placed_at"`
	UpdatedAt       *time.Time  `json:"updated_at,omitempty"`
}

// ── Category types ────────────────────────────────────────────────────────────

// Category is a self-referential tree node.
// Children []*Category creates a recursive schema; swag3 breaks the cycle
// by emitting a $ref on the second encounter of the same type.
type Category struct {
	ID       int64       `json:"id"                  example:"1"`
	Name     string      `json:"name"                example:"Dogs"`
	ParentID *int64      `json:"parent_id,omitempty" example:"0"`
	Children []*Category `json:"children,omitempty"`
}

// CreateCategoryRequest is the body for POST /categories.
type CreateCategoryRequest struct {
	Name     string `json:"name"               binding:"required" example:"Dogs"`
	ParentID *int64 `json:"parent_id,omitempty" example:"1"`
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// ListPets returns a paginated list of pets.
// Demonstrates composite-type annotation: PageResult{data=[]Pet}
// overrides the generic `data interface{}` field with a concrete []Pet.
//
// @Summary List pets (paginated)
// @Tags pets
// @ID listPets
// @Param status   query string  false "filter by status (available, pending, sold)"
// @Param page     query integer false "page number (1-based, default 1)"
// @Param per_page query integer false "items per page (default 20, max 100)"
// @Success 200 {object} BaseResponse{data=PagedList{list=[]Pet}} "paginated pet list"
// @Failure 500 {object} BaseResponse                              "internal server error"
// @Router /pets [get]
func ListPets() {}

// GetPet returns a single pet by ID.
//
// @Summary Get a pet by ID
// @Tags pets
// @ID getPet
// @Param id path integer true "pet ID"
// @Success 200 {object} BaseResponse{data=Pet} "the requested pet"
// @Failure 400 {object} BaseResponse           "invalid id"
// @Failure 404 {object} BaseResponse           "pet not found"
// @Router /pets/{id} [get]
func GetPet() {}

// CreatePet adds a new pet to the store.
//
// @Summary Create a new pet
// @Tags pets
// @ID createPet
// @Param body body CreatePetRequest true "pet to create"
// @Success 201 {object} BaseResponse{data=Pet} "created pet"
// @Failure 400 {object} BaseResponse           "invalid request body"
// @Security ApiKeyAuth
// @Router /pets [post]
func CreatePet() {}

// UpdatePet replaces an existing pet record.
//
// @Summary Update an existing pet
// @Tags pets
// @ID updatePet
// @Param id   path integer          true "pet ID"
// @Param body body CreatePetRequest true "updated pet data"
// @Success 200 {object} BaseResponse{data=Pet} "updated pet"
// @Failure 400 {object} BaseResponse           "invalid id or request body"
// @Failure 404 {object} BaseResponse           "pet not found"
// @Security ApiKeyAuth
// @Router /pets/{id} [put]
func UpdatePet() {}

// DeletePet removes a pet from the store.
//
// @Summary Delete a pet
// @Tags pets
// @ID deletePet
// @Param id path integer true "pet ID"
// @Success 204 "pet deleted (no body)"
// @Failure 400 {object} BaseResponse "invalid id"
// @Failure 404 {object} BaseResponse "pet not found"
// @Security ApiKeyAuth
// @Router /pets/{id} [delete]
func DeletePet() {}

// SearchPets performs an advanced keyword + status search.
// Demonstrates function-local struct: SearchFilter is defined inside this
// function body. swag3 registers it automatically because this function
// carries a @Router annotation.
//
// @Summary Search pets by keyword, status, and tags
// @Tags pets
// @ID searchPets
// @Param body body   SearchFilter true "search criteria"
// @Success 200 {object} BaseResponse{data=PagedList{list=[]Pet}} "matching pets (single page)"
// @Failure 400 {object} BaseResponse                              "invalid search request"
// @Security ApiKeyAuth
// @Router /pets/search [post]
func SearchPets() {
	// SearchFilter is intentionally defined inside the function body to
	// demonstrate function-local struct support in swag3.
	type SearchFilter struct {
		Keywords []string `json:"keywords"`
		Statuses []string `json:"statuses"`
		Tags     []string `json:"tags,omitempty"`
	}
}

// GetPetEvents lists lifecycle events for a pet.
// Demonstrates PetEvent whose fields use stdlib (time.Time) and
// map (map[string]string) types that map to OpenAPI primitives.
//
// @Summary Get lifecycle events for a pet
// @Tags pets
// @ID getPetEvents
// @Param id         path  integer true  "pet ID"
// @Param event_type query string  false "filter by event type (created, updated, viewed, …)"
// @Success 200 {object} BaseResponse{data=[]PetEvent} "list of events"
// @Failure 400 {object} BaseResponse                  "invalid id"
// @Failure 404 {object} BaseResponse                  "pet not found"
// @Router /pets/{id}/events [get]
func GetPetEvents() {}

// UploadPetAvatar uploads a new avatar image for a pet (replaces any existing one).
// Demonstrates multipart/form-data upload with formData file parameter.
//
// @Summary Upload pet avatar
// @Tags pets
// @ID uploadPetAvatar
// @accept multipart/form-data
// @Param id   path     integer true "pet ID"
// @Param file formData file    true "avatar image (JPEG / PNG / GIF / WEBP, max 8 MiB)"
// @Success 200 {object} BaseResponse{data=UploadResult} "upload metadata"
// @Failure 400 {object} BaseResponse "invalid id or missing/oversized file"
// @Failure 404 {object} BaseResponse "pet not found"
// @Security ApiKeyAuth
// @Router /pets/{id}/avatar [post]
func UploadPetAvatar() {}

// DownloadPetAvatar streams the stored avatar image back to the caller.
// The response Content-Type matches the format of the uploaded file.
// Demonstrates a binary (application/octet-stream) download endpoint.
//
// @Summary Download pet avatar
// @Tags pets
// @ID downloadPetAvatar
// @produce application/octet-stream
// @Param id path integer true "pet ID"
// @Success 200 "avatar image bytes"
// @Failure 400 {object} BaseResponse "invalid id"
// @Failure 404 {object} BaseResponse "pet or avatar not found"
// @Router /pets/{id}/avatar [get]
func DownloadPetAvatar() {}

// ── Import (function-local struct demo) ───────────────────────────────────────

// ImportPets batch-imports pets from an external source.
// Demonstrates two function-local struct types (ImportItem, ImportRequest)
// within a single annotated operation; swag3 registers both because their
// enclosing function carries a @Router annotation.
//
// @Summary Batch-import pets
// @Tags pets
// @ID importPets
// @Param body body ImportRequest true "import payload"
// @Success 200 {object} BaseResponse{data=[]Pet} "successfully imported pets"
// @Failure 400 {object} BaseResponse             "invalid request body"
// @Security ApiKeyAuth
// @Router /pets/import [post]
func ImportPets() {
	// ImportItem describes a single pet to import.
	type ImportItem struct {
		Name   string   `json:"name"   binding:"required"`
		Status string   `json:"status"`
		Tags   []string `json:"tags,omitempty"`
	}

	// ImportRequest is the top-level body for the import endpoint.
	type ImportRequest struct {
		Source string       `json:"source" binding:"required"` // "csv" | "json" | "api"
		DryRun bool         `json:"dry_run"`
		Items  []ImportItem `json:"items" binding:"required"`
	}
}

// ── Order handlers (nested struct + stdlib time.Time + cross-file reference) ──

// CreateOrder creates a new order.
// Demonstrates a deeply nested request body:
//   - ShippingAddress Address     inline struct (required)
//   - BillingAddress  *Address    nullable pointer — same struct, different schema nullability
//   - Items           []OrderItem array of structs with their own required fields
//   - PlacedAt        time.Time   stdlib → OpenAPI string/date-time
//
// Address and OrderItem are defined in types_order.go (separate file), showing
// that swag3 resolves cross-file $ref during directory-level parsing.
//
// @Summary Create an order
// @Tags orders
// @ID createOrder
// @Param body body CreateOrderRequest true "order payload"
// @Success 201 {object} BaseResponse{data=Order} "created order"
// @Failure 400 {object} BaseResponse             "invalid request body"
// @Security ApiKeyAuth
// @Router /orders [post]
func CreateOrder() {}

// GetOrder retrieves a single order by ID.
//
// @Summary Get an order by ID
// @Tags orders
// @ID getOrder
// @Param id path integer true "order ID"
// @Success 200 {object} BaseResponse{data=Order} "the order"
// @Failure 400 {object} BaseResponse             "invalid id"
// @Failure 404 {object} BaseResponse             "order not found"
// @Router /orders/{id} [get]
func GetOrder() {}

// ── Category handlers (recursive / self-referential schema) ───────────────────

// CreateCategory adds a category node to the tree.
// Category.Children []*Category creates a recursive schema; swag3 detects
// the cycle and resolves it with a $ref rather than expanding infinitely.
//
// @Summary Create a category
// @Tags categories
// @ID createCategory
// @Param body body CreateCategoryRequest true "category payload"
// @Success 201 {object} BaseResponse{data=Category} "created category"
// @Failure 400 {object} BaseResponse                "invalid request body"
// @Failure 404 {object} BaseResponse                "parent category not found"
// @Security ApiKeyAuth
// @Router /categories [post]
func CreateCategory() {}

// GetCategory returns a category with its full Children subtree.
// The recursive $ref in the Children field means the response can represent
// an arbitrary-depth tree without any special handling in the OpenAPI spec.
//
// @Summary Get a category (with children tree)
// @Tags categories
// @ID getCategory
// @Param id path integer true "category ID"
// @Success 200 {object} BaseResponse{data=Category} "category with children"
// @Failure 400 {object} BaseResponse                "invalid id"
// @Failure 404 {object} BaseResponse                "category not found"
// @Router /categories/{id} [get]
func GetCategory() {}
